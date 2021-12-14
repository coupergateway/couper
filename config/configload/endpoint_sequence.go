package configload

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
)

// addSequenceDeps collects possible dependencies from variables.
func addSequenceDeps(names map[string]struct{}, endpoint *config.Endpoint) {
	var items []config.SequenceItem
	for _, conf := range endpoint.Proxies {
		items = append(items, conf)
	}
	for _, conf := range endpoint.Requests {
		items = append(items, conf)
	}

	for _, seqItem := range items {
		allExpressions := body.CollectExpressions(seqItem.HCLBody())
		for _, expr := range allExpressions {
			for _, traversal := range expr.Variables() {
				if traversal.RootName() != "backend_responses" || len(traversal) < 2 {
					continue
				}

				// do we have a ref ?
				for _, t := range traversal[1:] {
					tr, ok := t.(hcl.TraverseAttr)
					if !ok {
						continue
					}

					_, ok = names[tr.Name]
					if !ok {
						continue
					}

					for _, i := range items {
						if i.GetName() != tr.Name || i == seqItem {
							continue
						}
						seqItem.Add(i)
						break
					}
				}
			}
		}
	}
}
