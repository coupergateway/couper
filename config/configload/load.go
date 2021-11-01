package configload

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
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
	definitions  = "definitions"
	errorHandler = "error_handler"
	nameLabel    = "name"
	oauth2       = "oauth2"
	proxy        = "proxy"
	request      = "request"
	server       = "server"
	settings     = "settings"
	// defaultNameLabel maps the the hcl label attr 'name'.
	defaultNameLabel = "default"
)

var regexProxyRequestLabel = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var envContext *hcl.EvalContext
var configBytes []byte

func init() {
	envContext = eval.NewContext(nil, nil).HCLContext()
}

// SetWorkingDirectory sets the working directory to the given configuration file path.
func SetWorkingDirectory(configFile string) (string, error) {
	if err := os.Chdir(filepath.Dir(configFile)); err != nil {
		return "", err
	}

	return os.Getwd()
}

func LoadFile(filePath string, verifyOnly bool) (*config.Couper, error) {
	_, err := SetWorkingDirectory(filePath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(filePath)

	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return LoadBytes(src, filename, verifyOnly)
}

func LoadBytes(src []byte, filename string, verifyOnly bool) (*config.Couper, error) {
	hclBody, diags := parser.Load(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	return LoadConfig(hclBody, src, filename, verifyOnly)
}

func LoadConfig(body hcl.Body, src []byte, filename string, verifyOnly bool) (*config.Couper, error) {
	if diags := ValidateConfigSchema(body, &config.Couper{}); diags.HasErrors() || verifyOnly {
		return nil, diags
	}

	defaultsBlock := &config.DefaultsBlock{}
	if diags := gohcl.DecodeBody(body, nil, defaultsBlock); diags.HasErrors() {
		return nil, diags
	}

	defaults := config.DefaultSettings
	defaults.AcceptForwarded = &config.AcceptForwarded{}

	evalContext := eval.NewContext(src, defaultsBlock.Defaults)
	envContext = evalContext.HCLContext()

	couperConfig := &config.Couper{
		Bytes:       src,
		Context:     evalContext,
		Definitions: &config.Definitions{},
		Defaults:    defaultsBlock.Defaults,
		Filename:    filename,
		Settings:    &defaults,
	}

	configBytes = src[:]

	schema, _ := gohcl.ImpliedBodySchema(couperConfig)
	content, diags := body.Content(schema)
	if content == nil {
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	// Read possible reference definitions first. Those are the
	// base for refinement merges during server block read out.
	var definedBackends Backends

	for _, outerBlock := range content.Blocks {
		switch outerBlock.Type {
		case definitions:
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return nil, diags
			}

			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					name := be.Labels[0]
					ref, _ := definedBackends.WithName(name)
					if ref != nil {
						return nil, hcl.Diagnostics{&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  fmt.Sprintf("duplicate backend name: %q", name),
							Subject:  &be.LabelRanges[0],
						}}
					}

					if err := uniqueAttributeKey(be.Body); err != nil {
						return nil, err
					}
					definedBackends = append(definedBackends, NewBackend(name, be.Body))
				}
			}

			if diags = gohcl.DecodeBody(leftOver, envContext, couperConfig.Definitions); diags.HasErrors() {
				return nil, diags
			}

			for _, oauth2Config := range couperConfig.Definitions.OAuth2AC {
				err := uniqueAttributeKey(oauth2Config.Remain)
				if err != nil {
					return nil, err
				}

				bodyContent, _, diags := oauth2Config.HCLBody().PartialContent(oauth2Config.Schema(true))
				if diags.HasErrors() {
					return nil, diags
				}
				oauth2Config.BodyContent = bodyContent

				oauth2Config.Backend, err = newBackend(definedBackends, oauth2Config)
				if err != nil {
					return nil, err
				}
			}

			for _, oidcConfig := range couperConfig.Definitions.OIDC {
				err := uniqueAttributeKey(oidcConfig.Remain)
				if err != nil {
					return nil, err
				}

				bodyContent, _, diags := oidcConfig.HCLBody().PartialContent(oidcConfig.Schema(true))
				if diags.HasErrors() {
					return nil, diags
				}
				oidcConfig.BodyContent = bodyContent

				oidcConfig.Backend, err = newBackend(definedBackends, oidcConfig)
				if err != nil {
					return nil, err
				}
			}

			for _, jwtConfig := range couperConfig.Definitions.JWT {
				err := uniqueAttributeKey(jwtConfig.Remain)
				if err != nil {
					return nil, err
				}

				if jwtConfig.JWKsURL != "" {
					bodyContent, _, diags := jwtConfig.HCLBody().PartialContent(jwtConfig.Schema(true))
					if diags.HasErrors() {
						return nil, diags
					}
					jwtConfig.BodyContent = bodyContent

					jwtConfig.Backend, err = newBackend(definedBackends, jwtConfig)
					if err != nil {
						return nil, err
					}

					jwtConfig.BackendName = ""
				}
				if err = jwtConfig.Check(); err != nil {
					return nil, errors.Configuration.Label(jwtConfig.Name).With(err)
				}
			}

			acErrorHandler := collectErrorHandlerSetter(couperConfig.Definitions)
			if err := configureErrorHandler(acErrorHandler, definedBackends); err != nil {
				return nil, err
			}

		case settings:
			if diags = gohcl.DecodeBody(outerBlock.Body, envContext, couperConfig.Settings); diags.HasErrors() {
				return nil, diags
			}
			if err := couperConfig.Settings.SetAcceptForwarded(); err != nil {
				diag := &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("invalid accept_forwarded_url: %q", err),
					Subject:  &outerBlock.DefRange,
				}
				return nil, diag
			}
		}
	}

	// Prepare dynamic functions
	for _, profile := range couperConfig.Definitions.JWTSigningProfile {
		if profile.Headers != nil {
			expression, _ := profile.Headers.Value(nil)
			headers := seetie.ValueToMap(expression)

			var errorMessage string
			if _, exists := headers["alg"]; exists {
				errorMessage = `"alg" cannot be set via "headers"`
			} else if _, exists := headers["typ"]; exists {
				errorMessage = `"typ" cannot be set via "headers"`
			}

			if errorMessage != "" {
				err := fmt.Errorf(errorMessage)
				return nil, errors.Configuration.Label(profile.Name).With(err)
			}
		}

		key, err := reader.ReadFromAttrFile("jwt_signing_profile key", profile.Key, profile.KeyFile)
		if err != nil {
			return nil, errors.Configuration.Label(profile.Name).With(err)
		}
		profile.KeyBytes = key
	}

	for _, saml := range couperConfig.Definitions.SAML {
		metadata, err := reader.ReadFromFile("saml2 idp_metadata_file", saml.IdpMetadataFile)
		if err != nil {
			return nil, errors.Configuration.Label(saml.Name).With(err)
		}
		saml.MetadataBytes = metadata
	}

	jwtSigningConfigs := make(map[string]*lib.JWTSigningConfig)
	for _, profile := range couperConfig.Definitions.JWTSigningProfile {
		if _, exists := jwtSigningConfigs[profile.Name]; exists {
			return nil, errors.Configuration.Messagef("jwt_signing_profile block with label %s already defined", profile.Name)
		}
		signConf, err := lib.NewJWTSigningConfigFromJWTSigningProfile(profile)
		if err != nil {
			return nil, errors.Configuration.Label(profile.Name).With(err)
		}
		jwtSigningConfigs[profile.Name] = signConf
	}
	for _, jwt := range couperConfig.Definitions.JWT {
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

	couperConfig.Context = evalContext.
		WithJWTSigningConfigs(jwtSigningConfigs).
		WithOAuth2AC(couperConfig.Definitions.OAuth2AC).
		WithSAML(couperConfig.Definitions.SAML)

	// Read per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range bodyToContent(body).Blocks.OfType(server) {
		serverConfig := &config.Server{}
		if diags = gohcl.DecodeBody(serverBlock.Body, envContext, serverConfig); diags.HasErrors() {
			return nil, diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		// Read api blocks and merge backends with server and definitions backends.
		for _, apiBlock := range bodyToContent(serverConfig.Remain).Blocks.OfType(api) {
			apiConfig := &config.API{}
			if diags = gohcl.DecodeBody(apiBlock.Body, envContext, apiConfig); diags.HasErrors() {
				return nil, diags
			}

			if len(apiBlock.Labels) > 0 {
				apiConfig.Name = apiBlock.Labels[0]
			}

			err := refineEndpoints(definedBackends, apiConfig.Endpoints, true)
			if err != nil {
				return nil, err
			}

			apiConfig.CatchAllEndpoint = createCatchAllEndpoint()
			serverConfig.APIs = append(serverConfig.APIs, apiConfig)

			apiErrorHandler := collectErrorHandlerSetter(apiConfig)
			if err = configureErrorHandler(apiErrorHandler, definedBackends); err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		err := refineEndpoints(definedBackends, serverConfig.Endpoints, true)
		if err != nil {
			return nil, err
		}

		couperConfig.Servers = append(couperConfig.Servers, serverConfig)
	}

	if len(couperConfig.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing 'server' block")
	}

	return couperConfig, nil
}

// mergeBackendBodies appends the left side object with newly defined attributes or overrides already defined ones.
func mergeBackendBodies(definedBackends Backends, inline config.Inline) (hcl.Body, error) {
	var reference hcl.Body
	if beRef, ok := inline.(config.BackendReference); ok {
		r, err := getBackendReference(definedBackends, beRef)
		if err != nil {
			return nil, err
		}
		reference = r
	}

	content, _, diags := inline.HCLBody().PartialContent(inline.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	if content == nil {
		if reference != nil {
			return reference, nil
		}
		return nil, fmt.Errorf("configuration error: missing backend reference or inline definition")
	}

	// Apply current attributes to the referenced body.
	if len(content.Attributes) > 0 && reference != nil {
		reference = MergeBodies([]hcl.Body{reference, hclbody.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange,
		})})
	}

	var backendBlock *hcl.Block
	if backends := content.Blocks.OfType(backend); len(backends) > 0 {
		backendBlock = backends[0]
	} else {
		return reference, nil
	}

	// Case: `backend {}`, anonymous backend.
	if len(backendBlock.Labels) == 0 {
		return backendBlock.Body, nil
	}

	// Case: `backend "reference" {}`, referenced backend.
	refOverride, err := definedBackends.WithName(backendBlock.Labels[0])
	if err != nil {
		err.(hcl.Diagnostics)[0].Subject = &backendBlock.DefRange

		// Case: referenced backend is not defined in definitions.
		return nil, err
	}

	// link backend block name (label) to attribute 'name'
	if syntaxBody, ok := backendBlock.Body.(*hclsyntax.Body); ok {
		if refBody, ok := refOverride.(*hclsyntax.Body); ok {
			syntaxBody.Attributes[nameLabel] = refBody.Attributes[nameLabel]
		}
	}

	return MergeBodies([]hcl.Body{refOverride, backendBlock.Body}), nil
}

// getBackendReference tries to fetch a backend from `definitions`
// block by a reference name, e.g. `backend = "name"`.
func getBackendReference(definedBackends Backends, be config.BackendReference) (hcl.Body, error) {
	name := be.Reference()

	// backend string attribute just not set
	if name == "" {
		return nil, nil
	}

	reference, err := definedBackends.WithName(name)
	if err != nil {
		return nil, err // parse err
	}

	// a name is given but we have no definition
	if body, ok := be.(config.Inline); ok {
		if b := body.HCLBody(); reference == nil && b != nil {
			r := b.MissingItemRange()
			return nil, hcl.Diagnostics{&hcl.Diagnostic{
				Subject: &r,
				Summary: fmt.Sprintf("backend reference '%s' is not defined", name),
			}}
		}
	}

	return reference, nil
}

func refineEndpoints(definedBackends Backends, endpoints config.Endpoints, check bool) error {
	for _, endpoint := range endpoints {
		if err := uniqueAttributeKey(endpoint.Remain); err != nil {
			return err
		}

		if check && endpoint.Pattern == "" {
			var r hcl.Range
			if endpoint.Remain != nil {
				r = endpoint.Remain.MissingItemRange()
			}
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "endpoint: missing path pattern",
				Subject:  &r,
			}}
		}

		endpointContent := bodyToContent(endpoint.Remain)

		proxies := endpointContent.Blocks.OfType(proxy)
		requests := endpointContent.Blocks.OfType(request)

		if check && len(proxies)+len(requests) == 0 && endpoint.Response == nil {
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "missing 'default' proxy or request block, or a response definition",
				Subject:  &endpointContent.MissingItemRange,
			}}
		}

		proxyRequestLabelRequired := len(proxies)+len(requests) > 1

		for _, proxyBlock := range proxies {
			// TODO: refactor with request construction below // almost same ( later :-) )
			proxyConfig := &config.Proxy{}
			if diags := gohcl.DecodeBody(proxyBlock.Body, envContext, proxyConfig); diags.HasErrors() {
				return diags
			}
			if len(proxyBlock.Labels) > 0 {
				proxyConfig.Name = proxyBlock.Labels[0]
			}
			if proxyConfig.Name == "" {
				proxyConfig.Name = defaultNameLabel
			}

			wsEnabled, wsBody, wsErr := getWebsocketsConfig(proxyConfig)
			if wsErr != nil {
				return wsErr
			}

			if wsEnabled {
				if proxyConfig.Name != defaultNameLabel {
					return errors.Configuration.Message("websockets attribute or block is only allowed in a 'default' proxy block")
				}
				if proxyRequestLabelRequired || endpoint.Response != nil {
					return errors.Configuration.Message("websockets are allowed in the endpoint; other 'proxy', 'request' or 'response' blocks are not allowed")
				}

				if wsBody != nil {
					proxyBlock.Body = MergeBodies([]hcl.Body{proxyBlock.Body, wsBody})
				}
			}

			proxyConfig.Remain = proxyBlock.Body

			err := uniqueAttributeKey(proxyConfig.Remain)
			if err != nil {
				return err
			}

			proxyConfig.Backend, err = newBackend(definedBackends, proxyConfig)
			if err != nil {
				return err
			}

			endpoint.Proxies = append(endpoint.Proxies, proxyConfig)
		}

		for _, reqBlock := range requests {
			reqConfig := &config.Request{}
			if diags := gohcl.DecodeBody(reqBlock.Body, envContext, reqConfig); diags.HasErrors() {
				return diags
			}

			if len(reqBlock.Labels) > 0 {
				reqConfig.Name = reqBlock.Labels[0]
			}
			if reqConfig.Name == "" {
				reqConfig.Name = defaultNameLabel
			}

			// remap request specific names for headers and query to well known ones
			content, leftOvers, diags := reqBlock.Body.PartialContent(reqConfig.Schema(true))
			if diags.HasErrors() {
				return diags
			}

			if err := verifyBodyAttributes(content); err != nil {
				return err
			}

			renameAttribute(content, "headers", "set_request_headers")
			renameAttribute(content, "query_params", "set_query_params")

			reqConfig.Remain = MergeBodies([]hcl.Body{leftOvers, hclbody.New(content)})

			err := uniqueAttributeKey(reqConfig.Remain)
			if err != nil {
				return err
			}

			reqConfig.Backend, err = newBackend(definedBackends, reqConfig)
			if err != nil {
				return err
			}

			endpoint.Requests = append(endpoint.Requests, reqConfig)
		}

		if endpoint.Response != nil {
			content, _, _ := endpoint.Response.HCLBody().PartialContent(config.ResponseInlineSchema)
			_, existsBody := content.Attributes["body"]
			_, existsJsonBody := content.Attributes["json_body"]
			if existsBody && existsJsonBody {
				return hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "response can only have one of body or json_body attributes",
					Subject:  &content.Attributes["body"].Range,
				}}
			}
		}

		names := map[string]struct{}{}
		unique := map[string]struct{}{}
		itemRange := endpoint.Remain.MissingItemRange()
		for _, p := range endpoint.Proxies {
			names[p.Name] = struct{}{}

			if err := validLabelName(p.Name, &itemRange); err != nil {
				return err
			}

			if proxyRequestLabelRequired {
				if err := uniqueLabelName(unique, p.Name, &itemRange); err != nil {
					return err
				}
			}
		}

		for _, r := range endpoint.Requests {
			names[r.Name] = struct{}{}

			if err := validLabelName(r.Name, &itemRange); err != nil {
				return err
			}

			if proxyRequestLabelRequired {
				if err := uniqueLabelName(unique, r.Name, &itemRange); err != nil {
					return err
				}
			}
		}

		if _, ok := names[defaultNameLabel]; check && !ok && endpoint.Response == nil {
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a 'default' proxy or request definition, or a response block",
				Subject:  &itemRange,
			}}
		}

		epErrorHandler := collectErrorHandlerSetter(endpoint)
		if err := configureErrorHandler(epErrorHandler, definedBackends); err != nil {
			return err
		}
	}

	return nil
}

func getWebsocketsConfig(proxyConfig *config.Proxy) (bool, hcl.Body, error) {
	content, _, diags := proxyConfig.Remain.PartialContent(
		&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "websockets"}}},
	)
	if diags.HasErrors() {
		return false, nil, diags
	}

	if proxyConfig.Websockets != nil && len(content.Blocks.OfType("websockets")) > 0 {
		return false, nil, fmt.Errorf("either websockets attribute or block is allowed")
	}

	if proxyConfig.Websockets != nil {
		var body hcl.Body

		if *proxyConfig.Websockets {
			block := &hcl.Block{
				Type: "websockets",
				Body: EmptyBody(),
			}

			body = hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{block}})
		}

		return *proxyConfig.Websockets, body, nil
	}

	return len(content.Blocks) > 0, nil, nil
}

func verifyBodyAttributes(content *hcl.BodyContent) error {
	_, existsBody := content.Attributes["body"]
	_, existsFormBody := content.Attributes["form_body"]
	_, existsJsonBody := content.Attributes["json_body"]
	if existsBody && existsFormBody || existsBody && existsJsonBody || existsFormBody && existsJsonBody {
		rangeAttr := "body"
		if !existsBody {
			rangeAttr = "form_body"
		}
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "request can only have one of body, form_body or json_body attributes",
			Subject:  &content.Attributes[rangeAttr].Range,
		}}
	}
	return nil
}

func bodyToContent(body hcl.Body) *hcl.BodyContent {
	content := &hcl.BodyContent{
		MissingItemRange: body.MissingItemRange(),
	}
	b, ok := body.(*hclsyntax.Body)
	if !ok {
		return content
	}

	if len(b.Attributes) > 0 {
		content.Attributes = make(hcl.Attributes)
	}
	for name, attr := range b.Attributes {
		content.Attributes[name] = &hcl.Attribute{
			Name:      attr.Name,
			Expr:      attr.Expr,
			Range:     attr.Range(),
			NameRange: attr.NameRange,
		}
	}

	for _, block := range b.Blocks {
		content.Blocks = append(content.Blocks, &hcl.Block{
			Body:        block.Body,
			DefRange:    block.DefRange(),
			LabelRanges: block.LabelRanges,
			Labels:      block.Labels,
			Type:        block.Type,
			TypeRange:   block.TypeRange,
		})
	}

	return content
}

func contentByType(blockType string, body hcl.Body) (*hcl.BodyContent, error) {
	headerSchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: blockType},
		}}
	content, _, diags := body.PartialContent(headerSchema)
	if diags.HasErrors() {
		derr := diags.Errs()[0].(*hcl.Diagnostic)
		if derr.Summary == "Extraneous label for "+blockType { // retry with label
			headerSchema.Blocks[0].LabelNames = []string{nameLabel}
			content, _, diags = body.PartialContent(headerSchema)
			if diags.HasErrors() { // due to interface nil check, do not return empty diags
				return nil, diags
			}
			return content, nil
		}
		return nil, diags
	}
	return content, nil
}

func newBackend(definedBackends Backends, inlineConfig config.Inline) (hcl.Body, error) {
	bend, err := mergeBackendBodies(definedBackends, inlineConfig)
	if err != nil {
		return nil, err
	}

	if bend == nil {
		// Create a default backend
		bend = hclbody.New(&hcl.BodyContent{
			Attributes: map[string]*hcl.Attribute{
				"name": {
					Name: "name",
					Expr: &hclsyntax.LiteralValueExpr{
						Val: cty.StringVal(defaultNameLabel),
					},
				},
			},
		})
	}

	oauth2Backend, err := newOAuthBackend(definedBackends, bend)
	if err != nil {
		return nil, err
	}

	if oauth2Backend != nil {
		wrapped := hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{
			{Type: oauth2, Body: hclbody.New(&hcl.BodyContent{Blocks: []*hcl.Block{
				{Type: backend, Body: oauth2Backend},
			}})},
		}})
		bend = MergeBodies([]hcl.Body{bend, wrapped})
	}

	diags := uniqueAttributeKey(bend)
	return bend, diags
}

func createCatchAllEndpoint() *config.Endpoint {
	responseBody := hclbody.New(&hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{
			"status": {
				Name: "status",
				Expr: &hclsyntax.LiteralValueExpr{
					Val: cty.NumberIntVal(http.StatusNotFound),
				},
			},
		},
	})

	return &config.Endpoint{
		Pattern: "/**",
		Remain:  hclbody.New(&hcl.BodyContent{}),
		Response: &config.Response{
			Remain: responseBody,
		},
	}
}

func newOAuthBackend(definedBackends Backends, parent hcl.Body) (hcl.Body, error) {
	innerContent, err := contentByType(oauth2, parent)
	if err != nil {
		return nil, err
	}

	oauthBlocks := innerContent.Blocks.OfType(oauth2)
	if len(oauthBlocks) == 0 {
		return nil, nil
	}

	backendContent, err := contentByType(backend, oauthBlocks[0].Body)
	if err != nil {
		return nil, err
	}

	beConfig := &config.Backend{Remain: hclbody.New(backendContent)}

	attrs, _ := oauthBlocks[0].Body.JustAttributes()
	if attrs != nil && attrs["backend"] != nil {
		val, _ := attrs["backend"].Expr.Value(nil)

		if ref := seetie.ValueToString(val); ref != "" {
			beConfig.Name = ref
		}
	}

	oauthBackend, err := mergeBackendBodies(definedBackends, beConfig)
	if err != nil {
		return nil, err
	}

	return newBackend(definedBackends, &config.OAuth2ReqAuth{Remain: hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{Type: backend, Body: oauthBackend},
		},
	})})
}

func renameAttribute(content *hcl.BodyContent, old, new string) {
	if attr, ok := content.Attributes[old]; ok {
		attr.Name = new
		content.Attributes[new] = attr
		delete(content.Attributes, old)
	}
}
