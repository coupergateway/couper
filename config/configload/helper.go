package configload

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/sequence"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

type helper struct {
	config       *config.Couper
	context      *hcl.EvalContext
	content      *hcl.BodyContent
	defsBackends map[string]hcl.Body
}

// newHelper creates a container with some methods to keep things simple here and there.
func newHelper(body hcl.Body, src [][]byte) (*helper, error) {
	defaultsBlock := &config.DefaultsBlock{}
	if diags := gohcl.DecodeBody(body, nil, defaultsBlock); diags.HasErrors() {
		return nil, diags
	}

	defSettings := config.DefaultSettings

	couperConfig := &config.Couper{
		Context:     eval.NewContext(src, defaultsBlock.Defaults),
		Definitions: &config.Definitions{},
		Defaults:    defaultsBlock.Defaults,
		Settings:    &defSettings,
	}

	schema, _ := gohcl.ImpliedBodySchema(couperConfig)
	content, diags := body.Content(schema)
	if content == nil { // reference diags only for missing content, due to optional server label
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	return &helper{
		config:       couperConfig,
		content:      content,
		context:      couperConfig.Context.(*eval.Context).HCLContext(),
		defsBackends: make(map[string]hcl.Body),
	}, nil
}

func (h *helper) addBackend(block *hcl.Block) error {
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

func (h *helper) configureDefinedBackends() error {
	backendNames, err := h.resolveBackendDeps()
	if err != nil {
		return err
	}

	for _, name := range backendNames {
		b := h.defsBackends[name]
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
		if err := ac.Prepare(func(attr string, attrVal string, i config.Inline) (hcl.Body, error) {
			return PrepareBackend(h, attr, attrVal, i) // wrap helper
		}); err != nil {
			return err
		}
	}
	return nil
}

// resolveBackendDeps returns defined backends ordered by reference. Referenced ones needs to configured first.
func (h *helper) resolveBackendDeps() (uniqueItems []string, err error) {
	// collect referenced backends
	refs := make(map[string][]string)
	h.collectBackendDeps(refs)
	// built up deps
	refPtr := map[string]*sequence.Item{}
	for name := range refs {
		parent := &sequence.Item{Name: name}
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
			p, _ := refPtr[parent]
			if be, exist := refPtr[r]; exist {
				p.Add(be)
			} else {
				p.Add(&sequence.Item{Name: r})
			}
			defs = append(defs, p)
		}
	}

	items := sequence.Dependencies(defs)
	//fmt.Printf("%v\n", items)

	// do not forget the other ones
	var standalone []string
	for def, _ := range h.defsBackends {
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

	//fmt.Printf("%v\n", uniqueItems)
	return uniqueItems, err
}

func (h *helper) collectBackendDeps(refs map[string][]string) {
	for name, b := range h.defsBackends {
		refs[name] = nil
		content, _, _ := b.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{{Type: oauth2}}},
		)
		oaBlocks := content.Blocks.OfType(oauth2)
		for _, ob := range oaBlocks {
			osb, ok := ob.Body.(*hclsyntax.Body)
			if !ok {
				continue
			}

			for _, be := range osb.Attributes {
				if be.Name == backend {
					val, _ := be.Expr.Value(envContext)
					refs[name] = append(refs[name], val.AsString())
					break
				}
			}

			for _, block := range osb.Blocks {
				if block.Type != backend {
					continue
				}
				if len(block.Labels) > 0 {
					refs[name] = append(refs[name], block.Labels[0])
				}

				for _, subBlock := range block.Body.Blocks {
					if subBlock.Type == oauth2 {
						h.collectBackendDeps(refs)
						break
					}
				}
			}
		}
	}
}
