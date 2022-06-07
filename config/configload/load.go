package configload

import (
	"fmt"
	"io/ioutil"
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
	"github.com/avenga/couper/config/configload/collect"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

const (
	api          = "api"
	backend      = "backend"
	defaults     = "defaults"
	definitions  = "definitions"
	endpoint     = "endpoint"
	errorHandler = "error_handler"
	files        = "files"
	nameLabel    = "name"
	oauth2       = "oauth2"
	proxy        = "proxy"
	request      = "request"
	server       = "server"
	settings     = "settings"
	spa          = "spa"
	// defaultNameLabel maps the hcl label attr 'name'.
	defaultNameLabel = "default"
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

func updateContext(body hcl.Body, srcBytes [][]byte) hcl.Diagnostics {
	defaultsBlock := &config.DefaultsBlock{}
	if diags := gohcl.DecodeBody(body, nil, defaultsBlock); diags.HasErrors() {
		return diags
	}

	// We need the "envContext" to be able to resolve abs pathes in the config.
	defaultsConfig = defaultsBlock.Defaults
	evalContext = eval.NewContext(srcBytes, defaultsConfig)
	envContext = evalContext.HCLContext()

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

func LoadFiles(filesList []string) (*config.Couper, error) {
	var (
		srcBytes     [][]byte
		parsedBodies []*hclsyntax.Body
	)

	var loadedFiles []string
	for _, file := range filesList {
		filePath, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			// ReadDir ... returns a list ... sorted by filename.
			listing, err := ioutil.ReadDir(filePath)
			if err != nil {
				return nil, err
			}

			for _, item := range listing {
				if item.IsDir() || filepath.Ext(item.Name()) != ".hcl" {
					continue
				}

				filename := filepath.Join(filePath, item.Name())
				body, err := parseFile(filename, &srcBytes)
				if err != nil {
					return nil, err
				}

				loadedFiles = append(loadedFiles, filename)
				parsedBodies = append(parsedBodies, body)
			}
		} else {
			body, err := parseFile(filePath, &srcBytes)
			if err != nil {
				return nil, err
			}

			loadedFiles = append(loadedFiles, filePath)
			parsedBodies = append(parsedBodies, body)
		}
	}

	if len(srcBytes) == 0 {
		return nil, fmt.Errorf("missing configuration files")
	}

	defaultsBlock, err := mergeDefaults(parsedBodies)
	if err != nil {
		return nil, err
	}

	defs := &hclsyntax.Body{
		Blocks: hclsyntax.Blocks{defaultsBlock},
	}

	if diags := updateContext(defs, srcBytes); diags.HasErrors() {
		return nil, diags
	}

	for _, body := range parsedBodies {
		if err := absolutizePaths(body); err != nil {
			return nil, err
		}
	}

	settingsBlock := mergeSettings(parsedBodies)

	definitionsBlock, err := mergeDefinitions(parsedBodies)
	if err != nil {
		return nil, err
	}

	serverBlocks, err := mergeServers(parsedBodies)
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

	conf, err := LoadConfig(configBody, srcBytes[0], filesList[0])
	if err != nil {
		return nil, err
	}

	conf.Files = loadedFiles

	return conf, nil
}

func LoadFile(file string) (*config.Couper, error) {
	return LoadFiles([]string{file})
}

func LoadBytes(src []byte, filename string) (*config.Couper, error) {
	hclBody, diags := parser.Load(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	if diags = updateContext(hclBody, [][]byte{src}); diags.HasErrors() {
		return nil, diags
	}

	return LoadConfig(hclBody, src, filename)
}

func LoadConfig(body hcl.Body, src []byte, filename string) (*config.Couper, error) {
	var err error

	if diags := ValidateConfigSchema(body, &config.Couper{}); diags.HasErrors() {
		return nil, diags
	}

	helper, err := newHelper(body, src, filename)
	if err != nil {
		return nil, err
	}

	for _, outerBlock := range helper.content.Blocks {
		switch outerBlock.Type {
		case definitions:
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return nil, diags
			}

			// backends first
			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					if err = helper.addBackend(be); err != nil {
						return nil, err
					}
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

			if err = helper.config.Settings.SetAcceptForwarded(); err != nil {
				return nil, newDiagErr(&outerBlock.DefRange, fmt.Sprintf("invalid accept_forwarded_url: %q", err))
			}
		}
	}

	// Prepare dynamic functions
	for _, profile := range helper.config.Definitions.JWTSigningProfile {
		if profile.Headers != nil {
			expression, _ := profile.Headers.Value(nil)
			headers := seetie.ValueToMap(expression)

			var errorMessage string
			if _, exists := headers["alg"]; exists {
				errorMessage = `"alg" cannot be set via "headers"`
			} else if _, exists = headers["typ"]; exists {
				errorMessage = `"typ" cannot be set via "headers"`
			}

			if errorMessage != "" {
				return nil, errors.Configuration.Label(profile.Name).With(fmt.Errorf(errorMessage))
			}
		}

		key, err := reader.ReadFromAttrFile("jwt_signing_profile key", profile.Key, profile.KeyFile)
		if err != nil {
			return nil, errors.Configuration.Label(profile.Name).With(err)
		}
		profile.KeyBytes = key
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
		if _, exists := jwtSigningConfigs[profile.Name]; exists {
			return nil, errors.Configuration.Messagef("jwt_signing_profile block with label %s already defined", profile.Name)
		}
		signConf, err := lib.NewJWTSigningConfigFromJWTSigningProfile(profile)
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
			if _, exists := jwtSigningConfigs[jwt.Name]; exists {
				return nil, errors.Configuration.Messagef("jwt_signing_profile or jwt with label %s already defined", jwt.Name)
			}
			jwtSigningConfigs[jwt.Name] = signConf
		}
	}

	helper.config.Context = helper.config.Context.(*eval.Context).
		WithJWTSigningConfigs(jwtSigningConfigs).
		WithOAuth2AC(helper.config.Definitions.OAuth2AC).
		WithSAML(helper.config.Definitions.SAML)

	// Read per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range bodyToContent(body).Blocks.OfType(server) {
		serverConfig := &config.Server{}
		if diags := gohcl.DecodeBody(serverBlock.Body, helper.context, serverConfig); diags.HasErrors() {
			return nil, diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		// Read api blocks and merge backends with server and definitions backends.
		for _, apiBlock := range bodyToContent(serverConfig.Remain).Blocks.OfType(api) {
			apiConfig := &config.API{}
			if diags := gohcl.DecodeBody(apiBlock.Body, helper.context, apiConfig); diags.HasErrors() {
				return nil, diags
			}

			if len(apiBlock.Labels) > 0 {
				apiConfig.Name = apiBlock.Labels[0]
			}

			apiContent := bodyToContent(apiConfig.Remain)

			if apiConfig.AllowedMethods != nil && len(apiConfig.AllowedMethods) > 0 {
				if err = validMethods(apiConfig.AllowedMethods, &apiContent.Attributes["allowed_methods"].Range); err != nil {
					return nil, err
				}
			}

			rp := apiContent.Attributes["beta_required_permission"]
			if rp != nil {
				apiConfig.RequiredPermission = rp.Expr
			}

			err = refineEndpoints(helper, apiConfig.Endpoints, true)
			if err != nil {
				return nil, err
			}

			err = checkPermissionMixedConfig(apiConfig)
			if err != nil {
				return nil, err
			}

			apiConfig.CatchAllEndpoint = newCatchAllEndpoint()
			serverConfig.APIs = append(serverConfig.APIs, apiConfig)

			apiErrorHandler := collect.ErrorHandlerSetters(apiConfig)
			if err = configureErrorHandler(apiErrorHandler, helper); err != nil {
				return nil, err
			}
		}

		for _, spaBlock := range bodyToContent(serverConfig.Remain).Blocks.OfType(spa) {
			spaConfig := &config.Spa{}
			if diags := gohcl.DecodeBody(spaBlock.Body, helper.context, spaConfig); diags.HasErrors() {
				return nil, diags
			}

			if len(spaBlock.Labels) > 0 {
				spaConfig.Name = spaBlock.Labels[0]
			}

			serverConfig.SPAs = append(serverConfig.SPAs, spaConfig)
		}

		for _, filesBlock := range bodyToContent(serverConfig.Remain).Blocks.OfType(files) {
			filesConfig := &config.Files{}
			if diags := gohcl.DecodeBody(filesBlock.Body, helper.context, filesConfig); diags.HasErrors() {
				return nil, diags
			}

			if len(filesBlock.Labels) > 0 {
				filesConfig.Name = filesBlock.Labels[0]
			}

			serverConfig.Files = append(serverConfig.Files, filesConfig)
		}

		// standalone endpoints
		err = refineEndpoints(helper, serverConfig.Endpoints, true)
		if err != nil {
			return nil, err
		}

		helper.config.Servers = append(helper.config.Servers, serverConfig)
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
			if strings.HasPrefix(filePath, "file:") {
				filePath = filePath[5:]
			}
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
