package configload

import (
	"sort"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
)

// buildSequences collects possible dependencies from 'backend_responses' variable.
func buildSequences(names map[string]hcl.Body, endpoint *config.Endpoint) (err error) {
	sequences := map[string]*config.Sequence{}

	defer func() {
		if rc := recover(); rc != nil {
			err = rc.(error)
		}
	}()

	var sortedNames sort.StringSlice
	for name := range names {
		sortedNames = append(sortedNames, name)
	}
	sortedNames.Sort()

	for _, name := range sortedNames {
		b := names[name]
		refs := responseReferences(b)

		if len(refs) == 0 {
			continue
		}

		seq, exist := sequences[name]
		if !exist {
			seq = &config.Sequence{Name: name, BodyRange: getRange(b)}
			sequences[name] = seq
		}

		for _, r := range refs {
			ref, ok := sequences[r]
			if !ok {
				ref = &config.Sequence{Name: r, BodyRange: getRange(b)}
				sequences[r] = ref
			}
			// Do not add ourselves
			// Use case: modify response headers with current response
			if seq != ref {
				seq.Add(ref)
			}
		}
	}

	for _, s := range sequences {
		if !s.HasParent() {
			endpoint.Sequences = append(endpoint.Sequences, s)
		}
	}

	return err
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
