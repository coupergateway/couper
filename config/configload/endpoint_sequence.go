package configload

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
)

// buildSequences collects possible dependencies from 'backend_responses' variable.
func buildSequences(names map[string]hcl.Body, endpoint *config.Endpoint) {
	sequences := map[string]*config.Sequence{}

	for name, b := range names {
		refs := responseReferences(b)

		if len(refs) == 0 {
			continue
		}

		seq, exist := sequences[name]
		if !exist {
			seq = &config.Sequence{Name: name, BodyRange: b.MissingItemRange()}
			sequences[name] = seq
		}

		for _, r := range refs {
			ref, ok := sequences[r]
			if !ok {
				ref = &config.Sequence{Name: r, BodyRange: b.MissingItemRange()}
				sequences[r] = ref
			}
			seq.Add(ref)
		}
	}

	for _, s := range sequences {
		if !s.HasParent() {
			endpoint.Sequences = append(endpoint.Sequences, s)
		}
	}
}

func responseReferences(b hcl.Body) []string {
	var result []string
	unique := map[string]struct{}{}

	for _, expr := range body.CollectExpressions(b) {
		for _, traversal := range expr.Variables() {
			if traversal.RootName() != "backend_responses" || len(traversal) < 2 {
				continue
			}

			tr, ok := traversal[1].(hcl.TraverseAttr)
			if !ok {
				continue
			}

			if _, ok = unique[tr.Name]; !ok {
				unique[tr.Name] = struct{}{}
				result = append(result, tr.Name)
			}
		}
	}

	return result
}
