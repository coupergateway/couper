package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type Sequences []*Sequence

type Sequence struct {
	BodyRange *hcl.Range
	Name      string
	deps      Sequences
	parent    *Sequence
	seen      map[string]struct{}
}

func (s *Sequence) Add(ref *Sequence) {
	if strings.TrimSpace(ref.Name) == "" {
		return
	}

	ref.parent = s

	if s.hasSeen(ref.Name) { // collect names to populate error message
		refs := []string{ref.Name}
		p := s.parent
		for p != s {
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
}

// Deps returns sequence dependency in reversed order since they have to be solved first.
func (s *Sequence) Deps() Sequences {
	if len(s.deps) < 2 {
		return s.deps
	}

	var revert Sequences
	for i := len(s.deps); i > 0; i-- {
		revert = append(revert, s.deps[i-1])
	}
	return revert
}

func (s *Sequence) HasParent() bool {
	return s != nil && s.parent != nil
}

func (s *Sequence) hasSeen(name string) bool {
	if s == nil {
		return false
	}

	if s.seen == nil {
		s.seen = make(map[string]struct{})
	}

	if _, exist := s.seen[name]; exist {
		return true
	}

	s.seen[name] = struct{}{}

	if s.HasParent() && s.parent.hasSeen(name) {
		return true
	}

	return false
}

func ResolveSequence(item *Sequence, resolved, seen *[]string) {
	name := item.Name
	*seen = append(*seen, name)
	for _, dep := range item.Deps() {
		if !containsString(resolved, dep.Name) {
			if !containsString(seen, dep.Name) {
				ResolveSequence(dep, resolved, seen)
				continue
			}
		}
	}

	*resolved = append(*resolved, name)
}

// SequenceDependencies just collects the deps for filtering purposes.
func SequenceDependencies(items Sequences) (allDeps [][]string) {
	for _, item := range items {
		deps := make([]string, 0)
		seen := make([]string, 0)
		ResolveSequence(item, &deps, &seen)
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
