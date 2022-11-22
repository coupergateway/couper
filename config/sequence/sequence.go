package sequence

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type List []*Item

func NewBackendItem(name string) *Item {
	return &Item{
		Name:    name,
		backend: true,
	}
}

type Item struct {
	BodyRange *hcl.Range
	Name      string
	backend   bool
	deps      List
	parent    *Item
}

func (s *Item) Add(ref *Item) *Item {
	if strings.TrimSpace(ref.Name) == "" {
		return s
	}

	ref.parent = s

	if s.hasAncestor(ref.Name) { // collect names to populate error message
		refs := []string{ref.Name}
		p := s.parent
		for p != s {
			if p.parent == nil {
				break
			}

			p = p.parent
			name := p.Name
			deps := p.Deps()
			if len(deps) > 0 {
				name = deps[0].Name
			}
			refs = append(refs, name)
		}

		err := &hcl.Diagnostic{
			Detail: fmt.Sprintf("circular sequence reference: %s",
				strings.Join(append(refs, refs[0]), ",")),
			Severity: hcl.DiagError,
			Subject:  s.BodyRange,
			Summary:  "configuration error",
		}
		panic(err)
	}

	s.deps = append(s.deps, ref)
	return s
}

// Deps returns sequence dependency in reversed order since they have to be solved first.
func (s *Item) Deps() List {
	if len(s.deps) < 2 {
		return s.deps
	}

	var revert List
	for i := len(s.deps); i > 0; i-- {
		revert = append(revert, s.deps[i-1])
	}
	return revert
}

func (s *Item) HasParent() bool {
	return s != nil && s.parent != nil
}

func (s *Item) hasAncestor(name string) bool {
	if s == nil {
		return false
	}

	if !s.HasParent() {
		return false
	}

	if s.parent.Name == name {
		return true
	}

	return s.parent.hasAncestor(name)
}

func resolveSequence(item *Item, resolved, seen *[]string) {
	name := item.Name
	*seen = append(*seen, name)
	for _, dep := range item.Deps() {
		if !containsString(resolved, dep.Name) {
			if !containsString(seen, dep.Name) {
				resolveSequence(dep, resolved, seen)
				continue
			}
		}
	}

	*resolved = append(*resolved, name)
}

// Dependencies just collects the deps for filtering purposes.
func Dependencies(items List) (allDeps [][]string) {
	for _, item := range items {
		deps := make([]string, 0)
		seen := make([]string, 0)
		resolveSequence(item, &deps, &seen)
		allDeps = append(allDeps, deps)
	}
	return allDeps
}

func containsString(slice *[]string, needle string) bool {
	for _, n := range *slice {
		if n == needle {
			return true
		}
	}
	return false
}
