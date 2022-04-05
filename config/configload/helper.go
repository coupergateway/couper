package configload

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
)

type Helper struct {
	config       *config.Couper
	context      *hcl.EvalContext
	content      *hcl.BodyContent
	defsBackends map[string]hcl.Body
}

// newHelper creates a container with some methods to keep things simple here and there.
func newHelper(body hcl.Body, src []byte, filename, dirPath string) (*Helper, error) {
	defaultsBlock := &config.DefaultsBlock{}
	if diags := gohcl.DecodeBody(body, nil, defaultsBlock); diags.HasErrors() {
		return nil, diags
	}

	defSettings := config.DefaultSettings

	couperConfig := &config.Couper{
		Context:     eval.NewContext([][]byte{src}, defaultsBlock.Defaults),
		Definitions: &config.Definitions{},
		Defaults:    defaultsBlock.Defaults,
		Filename:    filename,
		Dirpath:     dirPath,
		Settings:    &defSettings,
	}

	schema, _ := gohcl.ImpliedBodySchema(couperConfig)
	content, diags := body.Content(schema)
	if content == nil { // reference diags only for missing content, due to optional server label
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	return &Helper{
		config:       couperConfig,
		content:      content,
		context:      couperConfig.Context.(*eval.Context).HCLContext(),
		defsBackends: make(map[string]hcl.Body),
	}, nil
}

func (h *Helper) addBackend(block *hcl.Block) error {
	name := block.Labels[0]

	if _, ok := h.defsBackends[name]; ok {
		return newDiagErr(&block.LabelRanges[0],
			fmt.Sprintf("duplicate backend name: %q", name))
	} else if strings.HasPrefix(name, "anonymous_") {
		return newDiagErr(&block.LabelRanges[0],
			fmt.Sprintf("backend name must not start with 'anonymous_': %q", name))
	}

	backendBody, err := newBodyWithName(name, block.Body)
	if err != nil {
		return err
	}

	h.defsBackends[name] = backendBody
	return nil
}

func (h *Helper) configureDefinedBackends() error {
	for name, b := range h.defsBackends {
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
	return nil
}

func (h *Helper) configureACBackends() error {
	var acs []config.BackendInitialization
	for _, ac := range h.config.Definitions.JWT {
		acs = append(acs, ac)
	}
	for _, ac := range h.config.Definitions.OAuth2AC {
		acs = append(acs, ac)
	}

	// TODO: remove with 1.8
	h.config.Definitions.OIDC = append(h.config.Definitions.OIDC, h.config.Definitions.BetaOIDC...)

	for _, ac := range append(h.config.Definitions.OIDC) {
		acs = append(acs, ac)
	}

	for _, ac := range acs {
		if err := ac.Prepare(func(attr string, attrVal string, i config.Inline) (hcl.Body, error) {
			return PrepareBackend(h, attr, attrVal, i) // wrap helper
		}); err != nil {
			return err
		}
	}
	return nil
}
