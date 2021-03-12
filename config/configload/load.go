package configload

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
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
	"github.com/avenga/couper/internal/seetie"
)

const (
	backend     = "backend"
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

	return LoadConfig(hclBody, src)
}

var envContext *hcl.EvalContext

func LoadConfig(body hcl.Body, src []byte) (*config.Couper, error) {
	defaults := config.DefaultSettings

	couperConfig := &config.Couper{
		Bytes:       src,
		Context:     eval.NewContext(src),
		Definitions: &config.Definitions{},
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
	couperConfig.Context = couperConfig.Context.WithJWTProfiles(couperConfig.Definitions.JWTSigningProfile)

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
	content, _, diags := body.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: backend},
		}})
	if diags.HasErrors() {
		return nil, diags
	}

	// read out possible attribute reference
	var name string
	if attr, ok := content.Attributes["backend"]; ok {
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
		// try to obtain proxy and request block with a chicken-and-egg situation:
		// hcl labels are required if set, to make them optional we must know the content
		// which could not unwrapped without label errors. We will handle this by block type
		// and may have to throw an error which hints the user to configure the file properly.
		endpointContent := &hcl.BodyContent{Attributes: make(hcl.Attributes)}
		for _, t := range []string{proxy, request} {
			c, err := contentByType(t, endpoint.Remain)
			if err != nil {
				return err
			}
			endpointContent.MissingItemRange = c.MissingItemRange
			endpointContent.Blocks = append(endpointContent.Blocks, c.Blocks...)
			for n, attr := range c.Attributes { // possible same key and content override, it's ok.
				endpointContent.Attributes[n] = attr
			}
		}

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
			proxyConfig.Backend, err = newBackend(definedBackends, proxyConfig, proxyConfig.URL)
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

			reqConfig.Remain = reqBlock.Body

			var err error
			reqConfig.Backend, err = newBackend(definedBackends, reqConfig, reqConfig.URL)
			if err != nil {
				return err
			}

			endpoint.Requests = append(endpoint.Requests, reqConfig)
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

func newBackend(definedBackends Backends, inlineConfig config.Inline, url string) (hcl.Body, error) {
	bend, err := mergeBackendBodies(definedBackends, inlineConfig)
	if err != nil {
		return nil, err
	}

	if url != "" {
		body, urlOrigin, err := newBackendFromURL(url)
		if err != nil {
			return nil, err
		}

		if bend == nil {
			bend = body
		} else {
			content, _, diags := bend.PartialContent(
				&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "origin"}}},
			)
			if diags.HasErrors() {
				return nil, diags
			}

			if attr, ok := content.Attributes["origin"]; ok {
				val, err := attr.Expr.Value(nil)
				if err != nil {
					return nil, err
				}
				beOrigin := seetie.ValueToString(val)

				if urlOrigin != beOrigin {
					r := inlineConfig.HCLBody().MissingItemRange()
					return nil, hcl.Diagnostics{&hcl.Diagnostic{
						Subject: &r,
						Summary: "The origin of 'url' and 'backend.origin' must be equal",
					}}
				}
			}

			bend = MergeBodies([]hcl.Body{bend, body})
		}
	}

	if err = validateOrigin(bend); err != nil {
		r := inlineConfig.HCLBody().MissingItemRange()
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &r,
			Summary: err.Error(),
		}}
	}

	return bend, nil
}

func newBackendFromURL(rawURL string) (hcl.Body, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}

	origin := u.Scheme + "://" + u.Host

	return hclbody.New(&hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{
			"name":   {Name: "name", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(defaultNameLabel)}},
			"origin": {Name: "origin", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(origin)}},
			"path":   {Name: "path", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(u.Path)}},
			"set_query_params": {
				Name: "set_query_params", Expr: &hclsyntax.LiteralValueExpr{Val: seetie.ValuesMapToValue(u.Query())},
			},
		},
	}), origin, nil
}

// validateOrigin checks at least for an origin attribute definition.
func validateOrigin(merged hcl.Body) error {
	if merged == nil {
		return fmt.Errorf("missing backend reference or definition")
	}

	content, _, diags := merged.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "origin"}}})
	if diags.HasErrors() {
		return diags
	}

	err := errors.New("missing backend.origin attribute")
	if content == nil {
		return err
	}

	_, ok := content.Attributes["origin"]
	if !ok {
		bodyRange := merged.MissingItemRange()
		if bodyRange.Filename == "<empty>" {
			return err
		}
		return hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &bodyRange,
			Summary: err.Error(),
		}}
	}
	return nil
}
