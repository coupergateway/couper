package configload

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

const (
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

type AccessControlSetter interface {
	Set(handler *config.ErrorHandler)
}

func init() {
	envContext = eval.NewContext(nil).HCLContext()
}

// SetWorkingDirectory sets the working directory to the given configuration file path.
func SetWorkingDirectory(configFile string) (string, error) {
	if err := os.Chdir(filepath.Dir(configFile)); err != nil {
		return "", err
	}

	return os.Getwd()
}

func LoadFile(filePath string) (*config.Couper, error) {
	_, err := SetWorkingDirectory(filePath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(filePath)

	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return LoadBytes(src, filename)
}

func LoadBytes(src []byte, filename string) (*config.Couper, error) {
	hclBody, diags := parser.Load(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	return LoadConfig(hclBody, src, filename)
}

func LoadConfig(body hcl.Body, src []byte, filename string) (*config.Couper, error) {
	defaults := config.DefaultSettings

	evalContext := eval.NewContext(src)
	envContext = evalContext.HCLContext()

	couperConfig := &config.Couper{
		Bytes:       src,
		Context:     evalContext,
		Definitions: &config.Definitions{},
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

			// access control - error_handler
			var acErrorHandler []AccessControlSetter
			for _, acConfig := range couperConfig.Definitions.BasicAuth {
				acErrorHandler = append(acErrorHandler, acConfig)
			}
			for _, acConfig := range couperConfig.Definitions.JWT {
				acErrorHandler = append(acErrorHandler, acConfig)
			}
			for _, acConfig := range couperConfig.Definitions.SAML {
				acErrorHandler = append(acErrorHandler, acConfig)
			}

			for _, ac := range acErrorHandler {
				acBody, ok := ac.(config.Body)
				if !ok {
					continue
				}
				acContent := bodyToContent(acBody.HCLBody())
				configuredLabels := map[string]struct{}{}
				for _, block := range acContent.Blocks.OfType(errorHandler) {
					errHandlerConf, err := newErrorHandlerConf(block.Labels, block.Body, definedBackends)
					if err != nil {
						return nil, err
					}

					for _, k := range errHandlerConf.Kinds {
						if _, exist := configuredLabels[k]; exist {
							return nil, hcl.Diagnostics{&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  fmt.Sprintf("duplicate error type registration: %q", k),
								Subject:  &block.LabelRanges[0],
							}}
						}

						if k != errors.Wildcard && !errors.IsKnown(k) {
							subjRange := block.DefRange
							if len(block.LabelRanges) > 0 {
								subjRange = block.LabelRanges[0]
							}
							diag := &hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  fmt.Sprintf("error type is unknown: %q", k),
								Subject:  &subjRange,
							}
							return nil, hcl.Diagnostics{diag}
						}

						configuredLabels[k] = struct{}{}
					}

					ac.Set(errHandlerConf)
				}

				if acDefault, has := ac.(config.ErrorHandlerGetter); has {
					defaultHandler := acDefault.DefaultErrorHandler()
					_, exist := configuredLabels[errors.Wildcard]
					if !exist {
						for _, kind := range defaultHandler.Kinds {
							_, exist = configuredLabels[kind]
							if exist {
								break
							}
						}
					}

					if !exist {
						ac.Set(acDefault.DefaultErrorHandler())
					}
				}
			}

		case settings:
			if diags = gohcl.DecodeBody(outerBlock.Body, envContext, couperConfig.Settings); diags.HasErrors() {
				return nil, diags
			}
		}
	}

	for _, outerBlock := range content.Blocks {
		switch outerBlock.Type {
		case definitions:
			definitionsContent := bodyToContent(outerBlock.Body)
			oauth2Blocks := definitionsContent.Blocks.OfType(oauth2)
			couperConfig.Definitions.OAuth2AC = []*config.OAuth2AC{}
			for _, oauth2Block := range oauth2Blocks {
				oauth2Config := &config.OAuth2AC{}
				if diags := gohcl.DecodeBody(oauth2Block.Body, envContext, oauth2Config); diags.HasErrors() {
					return nil, diags
				}
				if len(oauth2Block.Labels) > 0 {
					oauth2Config.Name = oauth2Block.Labels[0]
				}
				if oauth2Config.Name == "" {
					oauth2Config.Name = defaultNameLabel
				}

				oauth2Config.Remain = oauth2Block.Body

				err := uniqueAttributeKey(oauth2Config.Remain)
				if err != nil {
					return nil, err
				}

				oauth2Config.Backend, err = newBackend(definedBackends, oauth2Config)
				if err != nil {
					return nil, err
				}

				couperConfig.Definitions.OAuth2AC = append(couperConfig.Definitions.OAuth2AC, oauth2Config)
			}
		}
	}

	// Prepare dynamic functions
	couperConfig.Context = evalContext.
		WithJWTProfiles(couperConfig.Definitions.JWTSigningProfile).
		WithSAML(couperConfig.Definitions.SAML)

	// Read per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range content.Blocks.OfType(server) {
		serverConfig := &config.Server{}
		if diags = gohcl.DecodeBody(serverBlock.Body, envContext, serverConfig); diags.HasErrors() {
			return nil, diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		// Read api blocks and merge backends with server and definitions backends.
		for _, apiBlock := range serverConfig.APIs {
			err := refineEndpoints(definedBackends, apiBlock.Endpoints, true)
			if err != nil {
				return nil, err
			}

			apiBlock.CatchAllEndpoint = createCatchAllEndpoint()
		}

		// standalone endpoints
		err := refineEndpoints(definedBackends, serverConfig.Endpoints, true)
		if err != nil {
			return nil, err
		}

		couperConfig.Servers = append(couperConfig.Servers, serverConfig)
	}

	if len(couperConfig.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing server definition")
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
	}

	return nil
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

	oauthBackend, err := mergeBackendBodies(definedBackends, &config.Backend{Remain: hclbody.New(backendContent)})
	if err != nil {
		return nil, err
	}

	return newBackend(definedBackends, &config.OAuth2{Remain: hclbody.New(&hcl.BodyContent{
		Blocks: []*hcl.Block{
			{Type: backend, Body: oauthBackend},
		},
	})})
}

func newErrorHandlerConf(kindLabels []string, body hcl.Body, definedBackends Backends) (*config.ErrorHandler, error) {
	var allKinds []string // Support for all events within one label separated by space

	for _, kinds := range kindLabels {
		all := strings.Split(kinds, " ")
		for _, a := range all {
			if a == "" {
				return nil, errors.Configuration.Messagef("invalid format: %v", kindLabels)
			}
		}
		allKinds = append(allKinds, all...)
	}
	if len(allKinds) == 0 {
		allKinds = append(allKinds, errors.Wildcard)
	}

	errHandlerConf := &config.ErrorHandler{Kinds: allKinds}
	if d := gohcl.DecodeBody(body, envContext, errHandlerConf); d.HasErrors() {
		return nil, d
	}

	ep := &config.Endpoint{
		ErrorFile: errHandlerConf.ErrorFile,
		Response:  errHandlerConf.Response,
		Remain:    body,
	}

	if err := refineEndpoints(definedBackends, config.Endpoints{ep}, false); err != nil {
		return nil, err
	}

	errHandlerConf.Requests = ep.Requests
	errHandlerConf.Proxies = ep.Proxies

	return errHandlerConf, nil
}

func renameAttribute(content *hcl.BodyContent, old, new string) {
	if attr, ok := content.Attributes[old]; ok {
		attr.Name = new
		content.Attributes[new] = attr
		delete(content.Attributes, old)
	}
}
