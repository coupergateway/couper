package test

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"

	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/eval"
)

func (h *Helper) NewInlineContext(inlineHCL string) hcl.Body {
	type hclBody struct {
		Inline hcl.Body `hcl:",remain"`
	}

	var remain hclBody
	h.Must(hclsimple.Decode(h.tb.Name()+".hcl", []byte(inlineHCL), eval.NewDefaultContext().HCLContext(), &remain))
	return body.MergeBodies(remain.Inline)
}
