package configload

import (
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config"
	hclbody "github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/config/configload/collect"
	"github.com/coupergateway/couper/config/reader"
	"github.com/coupergateway/couper/config/sequence"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/seetie"
)

type helper struct {
	config       *config.Couper
	context      *hcl.EvalContext
	content      *hcl.BodyContent
	defsBackends map[string]*hclsyntax.Body
}

// newHelper creates a container with some methods to keep things simple here and there.
func newHelper(body hcl.Body) (*helper, error) {
	couperConfig := &config.Couper{
		Context:     evalContext,
		Definitions: &config.Definitions{},
		Defaults:    defaultsConfig,
		Settings:    config.NewDefaultSettings(),
	}

	schema, _ := gohcl.ImpliedBodySchema(couperConfig)
	content, diags := body.Content(schema)
	if content == nil { // reference diags only for missing content, due to optional server label
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	return &helper{
		config:       couperConfig,
		content:      content,
		context:      evalContext.HCLContext(),
		defsBackends: make(map[string]*hclsyntax.Body),
	}, nil
}

func (h *helper) addBackend(block *hcl.Block) {
	name := block.Labels[0]

	backendBody := block.Body.(*hclsyntax.Body)
	setName(name, backendBody)

	h.defsBackends[name] = backendBody
}

func (h *helper) configureDefinedBackends() error {
	backendNames, err := h.resolveBackendDeps()
	if err != nil {
		return err
	}

	for _, name := range backendNames {
		b, set := h.defsBackends[name]
		if !set {
			return errors.Configuration.Messagef("referenced backend %q is not defined", name)
		}
		be, err := PrepareBackend(h, "_init", "", &config.Backend{Name: name, Remain: b})
		if err != nil {
			return err
		}
		h.config.Definitions.Backend = append(
			h.config.Definitions.Backend,
			&config.Backend{Remain: be, Name: name},
		)

		h.defsBackends[name] = be
	}
	return err
}

func (h *helper) configureACBackends() error {
	var acs []config.BackendInitialization
	for _, ac := range h.config.Definitions.JWT {
		acs = append(acs, ac)
	}
	for _, ac := range h.config.Definitions.OAuth2AC {
		acs = append(acs, ac)
	}

	for _, ac := range h.config.Definitions.OIDC {
		acs = append(acs, ac)
	}

	for _, ac := range acs {
		if err := ac.Prepare(func(attr string, attrVal string, b config.Body) (*hclsyntax.Body, error) {
			return PrepareBackend(h, attr, attrVal, b) // wrap helper
		}); err != nil {
			return err
		}
	}
	return nil
}

func (h *helper) configureBlocks() error {
	var err error

	for _, outerBlock := range h.content.Blocks {
		switch outerBlock.Type {
		case definitions:
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return diags
			}

			// backends first
			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					h.addBackend(be)
				}

				if err = h.configureDefinedBackends(); err != nil {
					return err
				}
			}

			// decode all other blocks into definition struct
			if diags = gohcl.DecodeBody(leftOver, h.context, h.config.Definitions); diags.HasErrors() {
				return diags
			}

			if err = h.configureACBackends(); err != nil {
				return err
			}

			acErrorHandler := collect.ErrorHandlerSetters(h.config.Definitions)
			if err = configureErrorHandler(acErrorHandler, h); err != nil {
				return err
			}

		case settings:
			if diags := gohcl.DecodeBody(outerBlock.Body, h.context, h.config.Settings); diags.HasErrors() {
				return diags
			}
		}
	}

	return nil
}

func (h *helper) configureJWTSigningProfile() *errors.Error {
	for _, profile := range h.config.Definitions.JWTSigningProfile {
		if profile.Headers != nil {
			expression, _ := profile.Headers.Value(nil)
			headers := seetie.ValueToMap(expression)

			if _, exists := headers["alg"]; exists {
				return errors.Configuration.Label(profile.Name).With(fmt.Errorf(`"alg" cannot be set via "headers"`))
			}
		}
	}

	return nil
}

func (h *helper) configureSAML() *errors.Error {
	for _, saml := range h.config.Definitions.SAML {
		metadata, err := reader.ReadFromFile("saml2 idp_metadata_file", saml.IdpMetadataFile)
		if err != nil {
			return errors.Configuration.Label(saml.Name).With(err)
		}

		saml.MetadataBytes = metadata
	}

	return nil
}

func (h *helper) configureJWTSigningConfig() (map[string]*lib.JWTSigningConfig, *errors.Error) {
	jwtSigningConfigs := make(map[string]*lib.JWTSigningConfig)

	for _, profile := range h.config.Definitions.JWTSigningProfile {
		signConf, err := lib.NewJWTSigningConfigFromJWTSigningProfile(profile, nil)
		if err != nil {
			return nil, errors.Configuration.Label(profile.Name).With(err)
		}

		jwtSigningConfigs[profile.Name] = signConf
	}

	for _, jwt := range h.config.Definitions.JWT {
		signConf, err := lib.NewJWTSigningConfigFromJWT(jwt)
		if err != nil {
			return nil, errors.Configuration.Label(jwt.Name).With(err)
		}

		if signConf != nil {
			jwtSigningConfigs[jwt.Name] = signConf
		}
	}

	return jwtSigningConfigs, nil
}

// Reads per server block and merge backend settings which results in a final server configuration.
func (h *helper) configureServers(body *hclsyntax.Body) error {
	var err error
	defsACs := h.getDefinedACs()

	for _, serverBlock := range hclbody.BlocksOfType(body, server) {
		serverConfig := &config.Server{}
		if diags := gohcl.DecodeBody(serverBlock.Body, h.context, serverConfig); diags.HasErrors() {
			return diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		if err = checkReferencedAccessControls(serverBlock.Body, serverConfig.AccessControl, serverConfig.DisableAccessControl, defsACs); err != nil {
			return err
		}

		for _, fileConfig := range serverConfig.Files {
			if err := checkReferencedAccessControls(fileConfig.HCLBody(), fileConfig.AccessControl, fileConfig.DisableAccessControl, defsACs); err != nil {
				return err
			}
		}

		for _, spaConfig := range serverConfig.SPAs {
			if err := checkReferencedAccessControls(spaConfig.HCLBody(), spaConfig.AccessControl, spaConfig.DisableAccessControl, defsACs); err != nil {
				return err
			}
		}

		err = h.configureAPIs(serverConfig.APIs, defsACs)
		if err != nil {
			return err
		}

		// Standalone endpoints
		err = refineEndpoints(h, serverConfig.Endpoints, true, defsACs)
		if err != nil {
			return err
		}

		h.config.Servers = append(h.config.Servers, serverConfig)
	}

	return nil
}

// Reads api blocks and merge backends with server and definitions backends.
func (h *helper) configureAPIs(apis config.APIs, defsACs map[string]struct{}) error {
	var err error

	for _, apiConfig := range apis {
		apiBody := apiConfig.HCLBody()

		if apiConfig.AllowedMethods != nil && len(apiConfig.AllowedMethods) > 0 {
			if err = validMethods(apiConfig.AllowedMethods, apiBody.Attributes["allowed_methods"]); err != nil {
				return err
			}
		}

		if err := checkReferencedAccessControls(apiBody, apiConfig.AccessControl, apiConfig.DisableAccessControl, defsACs); err != nil {
			return err
		}

		rp := apiBody.Attributes["required_permission"]
		if rp != nil {
			apiConfig.RequiredPermission = rp.Expr
		}

		err = refineEndpoints(h, apiConfig.Endpoints, true, defsACs)
		if err != nil {
			return err
		}

		err = checkPermissionMixedConfig(apiConfig)
		if err != nil {
			return err
		}

		apiConfig.CatchAllEndpoint = newCatchAllEndpoint()

		apiErrorHandler := collect.ErrorHandlerSetters(apiConfig)
		if err = configureErrorHandler(apiErrorHandler, h); err != nil {
			return err
		}
	}

	return nil
}

func (h *helper) configureJobs() error {
	var err error

	for _, job := range h.config.Definitions.Job {
		attrs := job.Remain.(*hclsyntax.Body).Attributes
		r := attrs["interval"].Expr.Range()

		job.IntervalDuration, err = config.ParseDuration("interval", job.Interval, -1)
		if err != nil {
			return newDiagErr(&r, err.Error())
		} else if job.IntervalDuration == -1 {
			return newDiagErr(&r, "invalid duration")
		}

		endpointConf := &config.Endpoint{
			Pattern:  job.Name, // for error messages
			Remain:   job.Remain,
			Requests: job.Requests,
		}

		err = refineEndpoints(h, config.Endpoints{endpointConf}, false, nil)
		if err != nil {
			return err
		}

		job.Endpoint = endpointConf
	}

	return nil
}

func (h *helper) configureBindAddresses() error {
	h.config.Settings.BindAddresses = make(map[string]string)

	if h.config.Settings.BindAddress == "" {
		h.config.Settings.BindAddress = "*"
	}

	for _, addr := range strings.Split(h.config.Settings.BindAddress, ",") {
		addr = strings.TrimSpace(addr)

		if addr == "*" {
			h.config.Settings.BindAddresses[""] = "tcp"

			return nil
		} else if addr == "::" {
			h.config.Settings.BindAddresses["[::]"] = "tcp6"
		} else {
			if net.ParseIP(addr) == nil {
				return fmt.Errorf("invalid bind address given: %q", addr)
			}

			if strings.Contains(addr, ":") {
				h.config.Settings.BindAddresses["["+addr+"]"] = "tcp6"
			} else {
				h.config.Settings.BindAddresses[addr] = "tcp4"
			}
		}
	}

	return nil
}

// resolveBackendDeps returns defined backends ordered by reference. Referenced ones need to be configured first.
func (h *helper) resolveBackendDeps() (uniqueItems []string, err error) {
	// collect referenced backends
	refs := make(map[string][]string)
	h.collectBackendDeps(refs)
	// built up deps
	refPtr := map[string]*sequence.Item{}
	for name := range refs {
		parent := sequence.NewBackendItem(name)
		refPtr[name] = parent
	}

	defer func() {
		if p := recover(); p != nil { // since we use sequence related logic, replace wording due to backend context here
			err = errors.Configuration.Message(strings.Replace(fmt.Sprintf("%s", p), "sequence ", "", 1))
		}
	}()

	var defs sequence.List
	for parent, ref := range refs {
		for _, r := range ref {
			p := refPtr[parent]
			if be, exist := refPtr[r]; exist {
				p.Add(be)
			} else {
				p.Add(sequence.NewBackendItem(r))
			}
			defs = append(defs, p)
		}
	}

	items := sequence.Dependencies(defs)

	// do not forget the other ones
	var standalone []string
	for def := range h.defsBackends {
		standalone = append(standalone, def)
	}
	items = append(items, standalone)

	// unique by name /w score (sort?) // TODO: MAY refine with scoring of appearance
	unique := make(map[string]int)
	for _, seqItem := range items {
		for _, name := range seqItem {
			if _, exist := unique[name]; !exist {
				unique[name] = 1
				uniqueItems = append(uniqueItems, name)
			} else {
				unique[name]++
			}
		}
	}

	return uniqueItems, err
}

func (h *helper) collectBackendDeps(refs map[string][]string) {
	for name, b := range h.defsBackends {
		refs[name] = nil
		oaBlocks := hclbody.BlocksOfType(b, oauth2)
		h.collectFromBlocks(oaBlocks, name, refs)
		trBlocks := hclbody.BlocksOfType(b, tokenRequest)
		h.collectFromBlocks(trBlocks, name, refs)
	}
}

func (h *helper) collectFromBlocks(authorizerBlocks hclsyntax.Blocks, name string, refs map[string][]string) {
	for _, ab := range authorizerBlocks {
		for _, be := range ab.Body.Attributes {
			if be.Name == backend {
				val, _ := be.Expr.Value(envContext)
				refs[name] = append(refs[name], val.AsString())
				break
			}
		}

		for _, block := range ab.Body.Blocks {
			if block.Type != backend {
				continue
			}
			if len(block.Labels) > 0 {
				refs[name] = append(refs[name], block.Labels[0])
			}

			for _, subBlock := range block.Body.Blocks {
				switch subBlock.Type {
				case oauth2, tokenRequest:
					h.collectBackendDeps(refs)
				}
			}
		}
	}
}

func (h *helper) getDefinedACs() map[string]struct{} {
	definitions := h.config.Definitions
	definedACs := make(map[string]struct{})

	for _, ac := range definitions.BasicAuth {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range definitions.JWT {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range definitions.OAuth2AC {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range definitions.OIDC {
		definedACs[ac.Name] = struct{}{}
	}
	for _, ac := range definitions.SAML {
		definedACs[ac.Name] = struct{}{}
	}

	return definedACs
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
