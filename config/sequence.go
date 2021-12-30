package config

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type Sequences []*Sequence

type Sequence struct {
	BodyRange hcl.Range
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

	if s.hasSeen(ref.Name) { // TODO: handle here instead of runtime pkg
		panic("cyclic" + s.Name + "|" + ref.Name)
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

	return false
}
