package configload

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/config/startup"
	"github.com/avenga/couper/eval"
)

const (
	backend     = "backend"
	oauth2      = "oauth2"
	definitions = "definitions"
	nameLabel   = "name"
	proxy       = "proxy"
	request     = "request"
	server      = "server"
	settings    = "settings"
	// defaultNameLabel maps the the hcl label attr 'name'.
	defaultNameLabel = "default"
)

var regexProxyRequestLabel = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var envContext *hcl.EvalContext

func init() {
	envContext = eval.NewContext(nil).HCLContext()
}

func LoadFile(filePath string) (*config.Couper, error) {
	_, err := startup.SetWorkingDirectory(filePath)
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

	couperConfig := &config.Couper{
		Bytes:       src,
		Context:     eval.NewContext(src),
		Definitions: &config.Definitions{},
		Filename:    filename,
		Settings:    &defaults,
	}

	envContext = couperConfig.Context.HCLContext()

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

					definedBackends = append(definedBackends, NewBackend(name, be.Body))
				}
			}

			if diags = gohcl.DecodeBody(leftOver, envContext, couperConfig.Definitions); diags.HasErrors() {
				return nil, diags
			}
		case settings:
			if diags = gohcl.DecodeBody(outerBlock.Body, envContext, couperConfig.Settings); diags.HasErrors() {
				return nil, diags
			}
		}
	}

	// Prepare dynamic functions
	couperConfig.Context = couperConfig.Context.WithJWTProfiles(couperConfig.Definitions.JWTSigningProfile).WithSAML(couperConfig.Definitions.SAML)

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
			err := refineEndpoints(definedBackends, apiBlock.Endpoints)
			if err != nil {
				return nil, err
			}

			apiBlock.CatchAllEndpoint = createCatchAllEndpoint()
		}

		// standalone endpoints
		err := refineEndpoints(definedBackends, serverConfig.Endpoints)
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
	reference, err := getBackendReference(definedBackends, inline.HCLBody())
	if err != nil {
		return nil, err
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
func getBackendReference(definedBackends Backends, body hcl.Body) (hcl.Body, error) {
	content := bodyToContent(body)

	// read out possible attribute reference
	var name string
	if attr, ok := content.Attributes[backend]; ok {
		val, valDiags := attr.Expr.Value(envContext)
		if valDiags.HasErrors() {
			return nil, valDiags
		}
		name = val.AsString()
	}

	// backend string attribute just not set
	if name == "" {
		return nil, nil
	}

	reference, err := definedBackends.WithName(name)
	if err != nil {
		return nil, err // parse err
	}

	// a name is given but we have no definition
	if reference == nil {
		r := body.MissingItemRange()
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &r,
			Summary: fmt.Sprintf("backend reference '%s' is not defined", name),
		}}
	}

	return reference, nil
}

func refineEndpoints(definedBackends Backends, endpoints config.Endpoints) error {
	for _, endpoint := range endpoints {
		endpointContent := bodyToContent(endpoint.Remain)

		proxies := endpointContent.Blocks.OfType(proxy)
		requests := endpointContent.Blocks.OfType(request)

		if len(proxies)+len(requests) == 0 && endpoint.Response == nil {
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

			var err error
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

			renameAttribute(content, "headers", "set_request_headers")
			renameAttribute(content, "query_params", "set_query_params")

			reqConfig.Remain = MergeBodies([]hcl.Body{leftOvers, hclbody.New(content)})

			var err error
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

		if _, ok := names[defaultNameLabel]; !ok && endpoint.Response == nil {
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a 'default' proxy or request definition, or a response block",
				Subject:  &itemRange,
			}}
		}
	}

	return nil
}

func validLabelName(name string, hr *hcl.Range) error {
	if !regexProxyRequestLabel.MatchString(name) {
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "proxy or request label contains unallowed character(s), allowed are 'a-z', 'A-Z', '0-9' and '_'",
			Subject:  hr,
		}}
	}
	return nil
}

func uniqueLabelName(unique map[string]struct{}, name string, hr *hcl.Range) error {
	if _, exist := unique[name]; exist {
		if name == defaultNameLabel {
			return hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "proxy and request labels are required and only one 'default' label is allowed",
				Subject:  hr,
			}}
		}
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("proxy and request labels are required and must be unique: %q", name),
			Subject:  hr,
		}}
	}
	unique[name] = struct{}{}
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

	return bend, nil
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
		Remain: hclbody.New(&hcl.BodyContent{}),
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

func renameAttribute(content *hcl.BodyContent, old, new string) {
	if attr, ok := content.Attributes[old]; ok {
		attr.Name = new
		content.Attributes[new] = attr
		delete(content.Attributes, old)
	}
}
