package configload

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	configfile "github.com/avenga/couper/config/configload/file"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

const (
	api             = "api"
	backend         = "backend"
	defaults        = "defaults"
	definitions     = "definitions"
	endpoint        = "endpoint"
	environment     = "environment"
	environmentVars = "environment_variables"
	errorHandler    = "error_handler"
	files           = "files"
	nameLabel       = "name"
	oauth2          = "oauth2"
	proxy           = "proxy"
	request         = "request"
	server          = "server"
	settings        = "settings"
	spa             = "spa"
	tls             = "tls"
	tokenRequest    = "beta_token_request"
)

var defaultsConfig *config.Defaults
var evalContext *eval.Context
var envContext *hcl.EvalContext
var pathBearingAttributesMap map[string]struct{}

func init() {
	pathBearingAttributes := []string{
		"bootstrap_file",
		"ca_file",
		"document_root",
		"error_file",
		"file",
		"htpasswd_file",
		"idp_metadata_file",
		"jwks_url",
		"key_file",
		"signing_key_file",
	}

	pathBearingAttributesMap = make(map[string]struct{})
	for _, attributeName := range pathBearingAttributes {
		pathBearingAttributesMap[attributeName] = struct{}{}
	}
}

func LoadFiles(filesList []string, env string) (*config.Couper, error) {
	configFiles, err := configfile.NewFiles(filesList)
	if err != nil {
		return nil, err
	}

	parsedBodies, srcBytes, err := parseFiles(configFiles)
	if err != nil {
		return nil, err
	}

	if len(srcBytes) == 0 {
		return nil, fmt.Errorf("missing configuration files")
	}

	errorBeforeRetry := preprocessEnvironmentBlocks(parsedBodies, env)

	if env == "" {
		settingsBlock := mergeSettings(parsedBodies)
		confSettings := &config.Settings{}
		if diags := gohcl.DecodeBody(settingsBlock.Body, nil, confSettings); diags.HasErrors() {
			return nil, diags
		}
		if confSettings.Environment != "" {
			return LoadFiles(filesList, confSettings.Environment)
		}
	}

	if errorBeforeRetry != nil {
		return nil, errorBeforeRetry
	}

	conf, err := bodiesToConfig(parsedBodies, srcBytes, env)
	if err != nil {
		return nil, err
	}
	conf.Files = configFiles

	return conf, nil
}

func LoadFile(file, env string) (*config.Couper, error) {
	return LoadFiles([]string{file}, env)
}

func LoadBytes(src []byte, filename string) (*config.Couper, error) {
	return LoadBytesEnv(src, filename, "")
}

func LoadBytesEnv(src []byte, filename, env string) (*config.Couper, error) {
	hclBody, err := parser.Load(src, filename)
	if err != nil {
		return nil, err
	}

	if err = validateBody(hclBody, false); err != nil {
		return nil, err
	}

	return bodiesToConfig([]*hclsyntax.Body{hclBody}, [][]byte{src}, env)
}

func loadConfig(body *hclsyntax.Body) (*config.Couper, error) {
	var (
		err error
		e   *errors.Error
	)

	if diags := validateConfigSchema(body, &config.Couper{}); diags.HasErrors() {
		return nil, diags
	}

	helper, err := newHelper(body)
	if err != nil {
		return nil, err
	}

	err = helper.configureBlocks()
	if err != nil {
		return nil, err
	}

	e = helper.configureJWTSigningProfile()
	if e != nil {
		return nil, e
	}

	e = helper.configureSAML()
	if e != nil {
		return nil, e
	}

	jwtSigningConfigs, e := helper.configureJWTSigningConfig()
	if e != nil {
		return nil, e
	}

	helper.config.Context = helper.config.Context.(*eval.Context).
		WithJWTSigningConfigs(jwtSigningConfigs).
		WithOAuth2AC(helper.config.Definitions.OAuth2AC).
		WithSAML(helper.config.Definitions.SAML)

	err = helper.configureServers(body)
	if err != nil {
		return nil, err
	}

	err = helper.configureJobs()
	if err != nil {
		return nil, err
	}

	if len(helper.config.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing 'server' block")
	}

	return helper.config, nil
}

func absolutizePaths(fileBody *hclsyntax.Body) error {
	visitor := func(node hclsyntax.Node) hcl.Diagnostics {
		attribute, ok := node.(*hclsyntax.Attribute)
		if !ok {
			return nil
		}

		_, exists := pathBearingAttributesMap[attribute.Name]
		if !exists {
			return nil
		}

		value, diags := attribute.Expr.Value(envContext)
		if diags.HasErrors() {
			return diags
		}

		filePath := value.AsString()
		basePath := attribute.SrcRange.Filename
		var absolutePath string
		if attribute.Name == "jwks_url" {
			if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
				return nil
			}

			filePath = strings.TrimPrefix(filePath, "file:")
			if path.IsAbs(filePath) {
				return nil
			}

			absolutePath = "file:" + filepath.ToSlash(path.Join(filepath.Dir(basePath), filePath))
		} else {
			if filepath.IsAbs(filePath) {
				return nil
			}
			absolutePath = filepath.Join(filepath.Dir(basePath), filePath)
		}

		attribute.Expr = &hclsyntax.LiteralValueExpr{
			Val:      cty.StringVal(absolutePath),
			SrcRange: attribute.SrcRange,
		}

		return nil
	}

	diags := hclsyntax.VisitAll(fileBody, visitor)
	if diags.HasErrors() {
		return diags
	}
	return nil
}

func updateContext(body hcl.Body, srcBytes [][]byte, environment string) hcl.Diagnostics {
	defaultsBlock := &config.DefaultsBlock{}
	// defaultsCtx is a temporary one to allow env variables and functions for defaults {}
	defaultsCtx := eval.NewContext(srcBytes, nil, environment).HCLContext()
	if diags := gohcl.DecodeBody(body, defaultsCtx, defaultsBlock); diags.HasErrors() {
		return diags
	}
	defaultsConfig = defaultsBlock.Defaults // global assign

	// We need the "envContext" to be able to resolve absolute paths in the config.
	evalContext = eval.NewContext(srcBytes, defaultsConfig, environment)
	envContext = evalContext.HCLContext() // global assign

	return nil
}

func bodiesToConfig(parsedBodies []*hclsyntax.Body, srcBytes [][]byte, env string) (*config.Couper, error) {
	defaultsBlock, err := mergeDefaults(parsedBodies)
	if err != nil {
		return nil, err
	}

	defs := &hclsyntax.Body{
		Blocks: hclsyntax.Blocks{defaultsBlock},
	}

	if diags := updateContext(defs, srcBytes, env); diags.HasErrors() {
		return nil, diags
	}

	for _, body := range parsedBodies {
		if err = absolutizePaths(body); err != nil {
			return nil, err
		}

		if err = validateBody(body, false); err != nil {
			return nil, err
		}
	}

	settingsBlock := mergeSettings(parsedBodies)

	definitionsBlock, proxies, err := mergeDefinitions(parsedBodies)
	if err != nil {
		return nil, err
	}

	serverBlocks, err := mergeServers(parsedBodies, proxies)
	if err != nil {
		return nil, err
	}

	configBlocks := serverBlocks
	configBlocks = append(configBlocks, definitionsBlock)
	configBlocks = append(configBlocks, defaultsBlock)
	configBlocks = append(configBlocks, settingsBlock)

	configBody := &hclsyntax.Body{
		Blocks: configBlocks,
	}

	if err = validateBody(configBody, len(parsedBodies) > 1); err != nil {
		return nil, err
	}

	conf, err := loadConfig(configBody)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
