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

	"github.com/coupergateway/couper/config"
	configfile "github.com/coupergateway/couper/config/configload/file"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
)

const (
	api                          = "api"
	backend                      = "backend"
	betaJob                      = "beta_job"
	defaults                     = "defaults"
	definitions                  = "definitions"
	endpoint                     = "endpoint"
	environment                  = "environment"
	environmentVars              = "environment_variables"
	errorHandler                 = "error_handler"
	files                        = "files"
	job                          = "job"
	nameLabel                    = "name"
	oauth2                       = "oauth2"
	proxy                        = "proxy"
	request                      = "request"
	server                       = "server"
	settings                     = "settings"
	spa                          = "spa"
	tls                          = "tls"
	tokenRequest                 = "beta_token_request"
	betaRateLimit                = "beta_rate_limit"
	throttle                     = "throttle"
	betaBackendRateLimitExceeded = "beta_backend_rate_limit_exceeded"
	backendThrottleExceeded      = "backend_throttle_exceeded"
)

var defaultsConfig *config.Defaults
var evalContext *eval.Context
var envContext *hcl.EvalContext
var pathBearingAttributesMap map[string]struct{}

func init() {
	pathBearingAttributes := []string{
		"bootstrap_file",
		"ca_certificate_file",
		"ca_file",
		"client_certificate_file",
		"client_private_key_file",
		"document_root",
		"error_file",
		"file",
		"htpasswd_file",
		"idp_metadata_file",
		"jwks_url",
		"key_file",
		"leaf_certificate_file",
		"permissions_map_file",
		"private_key_file",
		"public_key_file",
		"roles_map_file",
		"server_ca_certificate_file",
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
	deprecate(parsedBodies)

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

	conf.Files = append(configFiles, conf.Files...)

	return conf, nil
}

func loadConfig(body *hclsyntax.Body) (*config.Couper, error) {
	var (
		err error
		e   *errors.Error
	)

	if diags := validateConfigSchema(body, &config.Couper{}); diags.HasErrors() {
		return nil, diags
	}

	h, err := newHelper(body)
	if err != nil {
		return nil, err
	}

	err = h.configureBlocks()
	if err != nil {
		return nil, err
	}

	e = h.configureJWTSigningProfile()
	if e != nil {
		return nil, e
	}

	e = h.configureSAML()
	if e != nil {
		return nil, e
	}

	jwtSigningConfigs, e := h.configureJWTSigningConfig()
	if e != nil {
		return nil, e
	}

	h.config.Context = h.config.Context.(*eval.Context).
		WithJWTSigningConfigs(jwtSigningConfigs).
		WithOAuth2AC(h.config.Definitions.OAuth2AC)

	err = h.configureBindAddresses()
	if err != nil {
		return nil, e
	}

	err = h.configureServers(body)
	if err != nil {
		return nil, err
	}

	err = h.configureJobs()
	if err != nil {
		return nil, err
	}

	if len(h.config.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing 'server' block")
	}

	return h.config, nil
}

func absolutizePaths(fileBody *hclsyntax.Body) ([]configfile.File, error) {
	const watchFilePrefix = "COUPER-WATCH-FILE: "

	visitor := func(node hclsyntax.Node) hcl.Diagnostics {
		var watchFile string

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
				watchFile = filePath
			} else {
				watchFile = filepath.ToSlash(path.Join(filepath.Dir(basePath), filePath))

				absolutePath = "file:" + watchFile
			}
		} else {
			if filepath.IsAbs(filePath) {
				watchFile = filePath
			} else {
				watchFile = filepath.Join(filepath.Dir(basePath), filePath)

				absolutePath = watchFile
			}
		}

		if absolutePath != "" {
			attribute.Expr = &hclsyntax.LiteralValueExpr{
				Val:      cty.StringVal(absolutePath),
				SrcRange: attribute.SrcRange,
			}
		}

		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("%s%s", watchFilePrefix, watchFile),
		}}
	}

	diags := hclsyntax.VisitAll(fileBody, visitor)

	var (
		newDiags   hcl.Diagnostics
		watchFiles []configfile.File
	)

	for _, diag := range diags {
		if diag.Severity == hcl.DiagWarning && strings.HasPrefix(diag.Summary, watchFilePrefix) {
			watchFiles = append(watchFiles, configfile.File{
				Path: strings.TrimPrefix(diag.Summary, watchFilePrefix),
			})
		} else {
			newDiags.Append(diag)
		}
	}

	if newDiags.HasErrors() {
		return nil, newDiags
	}

	return watchFiles, nil
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

	var watchFiles configfile.Files

	for _, body := range parsedBodies {
		files, err := absolutizePaths(body)
		if err != nil {
			return nil, err
		}

		if err = validateBody(body, false); err != nil {
			return nil, err
		}

		watchFiles = append(watchFiles, files...)
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

	conf.Files = append(conf.Files, watchFiles...)

	return conf, nil
}
