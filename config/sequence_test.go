package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveSequence(t *testing.T) {

	b := (&Sequence{Name: "b"}).Add(&Sequence{Name: "c"})

	tests := []struct {
		name        string
		item        *Sequence
		expResolved []string
	}{
		{name: "order A", item: (&Sequence{Name: "test"}).
			Add((&Sequence{Name: "a"}).Add(b)).
			Add(b),
			expResolved: []string{"c", "b", "a", "test"}},
		{name: "order B", item: (&Sequence{Name: "test"}).
			Add(b).
			Add((&Sequence{Name: "a"}).Add(b)),
			expResolved: []string{"c", "b", "a", "test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			var resolved, seen []string
			ResolveSequence(tt.item, &resolved, &seen)
			if diff := cmp.Diff(tt.expResolved, resolved); diff != "" {
				st.Errorf("\ngot:\t%#v\ndiff: %s\nseen:\t%#v", resolved, diff, seen)
			}
		})
	}
}
