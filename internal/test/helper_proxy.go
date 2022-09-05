package test

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/eval"
)

func (h *Helper) NewInlineContext(inlineHCL string) *hclsyntax.Body {
	type hclBody struct {
		Inline hcl.Body `hcl:",remain"`
	}

	var remain hclBody
	h.Must(hclsimple.Decode(h.tb.Name()+".hcl", []byte(inlineHCL), eval.NewDefaultContext().HCLContext(), &remain))
	return remain.Inline.(*hclsyntax.Body)
}
