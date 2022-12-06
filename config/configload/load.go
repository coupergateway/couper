package configload

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/configload/collect"
	configfile "github.com/avenga/couper/config/configload/file"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
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
	oauth2          = "oauth2"
	plugin          = "plugin"
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

func parseFile(filePath string, srcBytes *[][]byte) (*hclsyntax.Body, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	*srcBytes = append(*srcBytes, src)

	parsed, diags := hclparse.NewParser().ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, diags
	}

	return parsed.Body.(*hclsyntax.Body), nil
}

func parseFiles(files configfile.Files) ([]*hclsyntax.Body, [][]byte, error) {
	var (
		srcBytes     [][]byte
		parsedBodies []*hclsyntax.Body
	)

	for _, file := range files {
		if file.IsDir {
			childBodies, bytes, err := parseFiles(file.Children)
			if err != nil {
				return nil, bytes, err
			}

			parsedBodies = append(parsedBodies, childBodies...)
			srcBytes = append(srcBytes, bytes...)
		} else {
			body, err := parseFile(file.Path, &srcBytes)
			if err != nil {
				return nil, srcBytes, err
			}
			parsedBodies = append(parsedBodies, body)
		}
	}

	return parsedBodies, srcBytes, nil
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

	conf, err := LoadConfig(configBody)
	if err != nil {
		return nil, err
	}

	return conf, nil
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

type testContent struct {
	filename string
	src      []byte
}

func loadTestContents(tcs []testContent) (*config.Couper, error) {
	var (
		parsedBodies []*hclsyntax.Body
		srcs         [][]byte
	)

	for _, tc := range tcs {
		hclBody, err := parser.Load(tc.src, tc.filename)
		if err != nil {
			return nil, err
		}

		parsedBodies = append(parsedBodies, hclBody)
		srcs = append(srcs, tc.src)
	}

	return bodiesToConfig(parsedBodies, srcs, "")
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

func LoadConfig(body *hclsyntax.Body) (*config.Couper, error) {
	var err error

	if diags := ValidateConfigSchema(body, &config.Couper{}); diags.HasErrors() {
		return nil, diags
	}

	helper, err := newHelper(body)
	if err != nil {
		return nil, err
	}

	for _, outerBlock := range helper.content.Blocks {
		switch outerBlock.Type {
		case definitions:
			if err = LoadPlugins(helper.context, outerBlock.Body); err != nil {
				return nil, err
			}

			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return nil, diags
			}

			// backends first
			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					helper.addBackend(be)
				}

				if err = helper.configureDefinedBackends(); err != nil {
					return nil, err
				}
			}

			// decode all other blocks into definition struct
			if diags = gohcl.DecodeBody(leftOver, helper.context, helper.config.Definitions); diags.HasErrors() {
				return nil, diags
			}

			if err = helper.configureACBackends(); err != nil {
				return nil, err
			}

			acErrorHandler := collect.ErrorHandlerSetters(helper.config.Definitions)
			if err = configureErrorHandler(acErrorHandler, helper); err != nil {
				return nil, err
			}

		case settings:
			if diags := gohcl.DecodeBody(outerBlock.Body, helper.context, helper.config.Settings); diags.HasErrors() {
				return nil, diags
			}
		}
	}

	// Prepare dynamic functions
	for _, profile := range helper.config.Definitions.JWTSigningProfile {
		if profile.Headers != nil {
			expression, _ := profile.Headers.Value(nil)
			headers := seetie.ValueToMap(expression)

			if _, exists := headers["alg"]; exists {
				return nil, errors.Configuration.Label(profile.Name).With(fmt.Errorf(`"alg" cannot be set via "headers"`))
			}
		}
	}

	for _, saml := range helper.config.Definitions.SAML {
		metadata, err := reader.ReadFromFile("saml2 idp_metadata_file", saml.IdpMetadataFile)
		if err != nil {
			return nil, errors.Configuration.Label(saml.Name).With(err)
		}
		saml.MetadataBytes = metadata
	}

	jwtSigningConfigs := make(map[string]*lib.JWTSigningConfig)
	for _, profile := range helper.config.Definitions.JWTSigningProfile {
		signConf, err := lib.NewJWTSigningConfigFromJWTSigningProfile(profile, nil)
		if err != nil {
			return nil, errors.Configuration.Label(profile.Name).With(err)
		}
		jwtSigningConfigs[profile.Name] = signConf
	}
	for _, jwt := range helper.config.Definitions.JWT {
		signConf, err := lib.NewJWTSigningConfigFromJWT(jwt)
		if err != nil {
			return nil, errors.Configuration.Label(jwt.Name).With(err)
		}
		if signConf != nil {
			jwtSigningConfigs[jwt.Name] = signConf
		}
	}

	helper.config.Context = helper.config.Context.(*eval.Context).
		WithJWTSigningConfigs(jwtSigningConfigs).
		WithOAuth2AC(helper.config.Definitions.OAuth2AC).
		WithSAML(helper.config.Definitions.SAML)

	definedACs := make(map[string]struct{})
	for _, ac := range helper.config.Definitions.BasicAuth {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range helper.config.Definitions.JWT {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range helper.config.Definitions.OAuth2AC {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range helper.config.Definitions.OIDC {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range helper.config.Definitions.SAML {
		definedACs[ac.Name] = struct{}{}
	}

	// Read per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range hclbody.BlocksOfType(body, server) {
		serverConfig := &config.Server{}
		if diags := gohcl.DecodeBody(serverBlock.Body, helper.context, serverConfig); diags.HasErrors() {
			return nil, diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		if err := checkReferencedAccessControls(serverBlock.Body, serverConfig.AccessControl, serverConfig.DisableAccessControl, definedACs); err != nil {
			return nil, err
		}

		for _, fileConfig := range serverConfig.Files {
			if err := checkReferencedAccessControls(fileConfig.HCLBody(), fileConfig.AccessControl, fileConfig.DisableAccessControl, definedACs); err != nil {
				return nil, err
			}
		}

		for _, spaConfig := range serverConfig.SPAs {
			if err := checkReferencedAccessControls(spaConfig.HCLBody(), spaConfig.AccessControl, spaConfig.DisableAccessControl, definedACs); err != nil {
				return nil, err
			}
		}

		// Read api blocks and merge backends with server and definitions backends.
		for _, apiConfig := range serverConfig.APIs {
			apiBody := apiConfig.HCLBody()

			if apiConfig.AllowedMethods != nil && len(apiConfig.AllowedMethods) > 0 {
				if err = validMethods(apiConfig.AllowedMethods, apiBody.Attributes["allowed_methods"]); err != nil {
					return nil, err
				}
			}

			if err := checkReferencedAccessControls(apiBody, apiConfig.AccessControl, apiConfig.DisableAccessControl, definedACs); err != nil {
				return nil, err
			}

			rp := apiBody.Attributes["required_permission"]
			if rp != nil {
				apiConfig.RequiredPermission = rp.Expr
			}

			err = refineEndpoints(helper, apiConfig.Endpoints, true, definedACs)
			if err != nil {
				return nil, err
			}

			err = checkPermissionMixedConfig(apiConfig)
			if err != nil {
				return nil, err
			}

			apiConfig.CatchAllEndpoint = newCatchAllEndpoint()

			apiErrorHandler := collect.ErrorHandlerSetters(apiConfig)
			if err = configureErrorHandler(apiErrorHandler, helper); err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		err = refineEndpoints(helper, serverConfig.Endpoints, true, definedACs)
		if err != nil {
			return nil, err
		}

		helper.config.Servers = append(helper.config.Servers, serverConfig)
	}

	for _, job := range helper.config.Definitions.Job {
		attrs := job.Remain.(*hclsyntax.Body).Attributes
		r := attrs["interval"].Expr.Range()

		job.IntervalDuration, err = config.ParseDuration("interval", job.Interval, -1)
		if err != nil {
			return nil, newDiagErr(&r, err.Error())
		} else if job.IntervalDuration == -1 {
			return nil, newDiagErr(&r, "invalid duration")
		}

		endpointConf := &config.Endpoint{
			Pattern:  job.Name, // for error messages
			Remain:   job.Remain,
			Requests: job.Requests,
		}

		err = refineEndpoints(helper, config.Endpoints{endpointConf}, false, nil)
		if err != nil {
			return nil, err
		}

		job.Endpoint = endpointConf
	}

	if len(helper.config.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing 'server' block")
	}

	return helper.config, nil
}

// checkPermissionMixedConfig checks whether, for api blocks with at least two endpoints,
// all endpoints in api have either
// a) no required permission set or
// b) required permission or disable_access_control set
func checkPermissionMixedConfig(apiConfig *config.API) error {
	if apiConfig.RequiredPermission != nil {
		// default for required permission: no mixed config
		return nil
	}

	l := len(apiConfig.Endpoints)
	if l < 2 {
		// too few endpoints: no mixed config
		return nil
	}

	countEpsWithPermission := 0
	countEpsWithPermissionOrDisableAC := 0
	for _, e := range apiConfig.Endpoints {
		if e.RequiredPermission != nil {
			// endpoint has required permission attribute set
			countEpsWithPermission++
			countEpsWithPermissionOrDisableAC++
		} else if e.DisableAccessControl != nil {
			// endpoint has didable AC attribute set
			countEpsWithPermissionOrDisableAC++
		}
	}

	if countEpsWithPermission == 0 {
		// no endpoints with required permission: no mixed config
		return nil
	}

	if l > countEpsWithPermissionOrDisableAC {
		return errors.Configuration.Messagef("api with label %q has endpoint without required permission", apiConfig.Name)
	}

	return nil
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
