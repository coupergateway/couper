package sequence

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveSequence(t *testing.T) {

	b := (&Item{Name: "b"}).Add(&Item{Name: "c"})

	tests := []struct {
		name        string
		item        *Item
		expResolved []string
	}{
		{name: "order A", item: (&Item{Name: "test"}).
			Add((&Item{Name: "a"}).Add(b)).
			Add(b),
			expResolved: []string{"c", "b", "a", "test"}},
		{name: "order B", item: (&Item{Name: "test"}).
			Add(b).
			Add((&Item{Name: "a"}).Add(b)),
			expResolved: []string{"c", "b", "a", "test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			var resolved, seen []string
			resolveSequence(tt.item, &resolved, &seen)
			if diff := cmp.Diff(tt.expResolved, resolved); diff != "" {
				st.Errorf("\ngot:\t%#v\ndiff: %s\nseen:\t%#v", resolved, diff, seen)
			}
		})
	}
}
