package sequence

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestItem_ResolveSequence(t *testing.T) {

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

func TestItem_Add_error(t *testing.T) {

	tests := []struct {
		name     string
		itemFunc func(a, b, c *Item) *Item
	}{
		{
			name: "circle 1",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(b)
				b.Add(a)
				return a
			},
		},
		{
			name: "circle 2",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(b)
				b.Add(c)
				c.Add(a)
				return a
			},
		},
		{
			name: "add self",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(a)
				return a
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			i := &Item{Name: "i"}
			j := &Item{Name: "j"}
			k := &Item{Name: "k"}

			defer func() { _ = recover() }()

			tt.itemFunc(i, j, k)

			st.Error("Missing panic")
		})
	}
}

func TestBackendItem_ResolveSequence(t *testing.T) {

	tests := []struct {
		name        string
		itemFunc    func(a, b, c *Item) *Item
		expResolved []string
	}{
		{
			name: "order A",
			itemFunc: func(a, b, c *Item) *Item {
				b.Add(c)
				a.Add(b)
				a.Add(c)
				return a
			},
			expResolved: []string{"vault", "as", "rs"},
		},
		{
			name: "order B",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(b)
				a.Add(c)
				b.Add(c)
				return a
			},
			expResolved: []string{"vault", "as", "rs"},
		},
		{
			name: "multiple",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(b)
				a.Add(b)
				return a
			},
			expResolved: []string{"as", "rs"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			var resolved, seen []string

			rs := NewBackendItem("rs")
			as := NewBackendItem("as")
			vault := NewBackendItem("vault")

			resolveSequence(tt.itemFunc(rs, as, vault), &resolved, &seen)
			if diff := cmp.Diff(tt.expResolved, resolved); diff != "" {
				st.Errorf("\ngot:\t%#v\ndiff: %s\nseen:\t%#v", resolved, diff, seen)
			}
		})
	}
}

func TestBackendItem_Add_error(t *testing.T) {

	tests := []struct {
		name     string
		itemFunc func(a, b, c *Item) *Item
	}{
		{
			name: "circle 1",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(b)
				b.Add(a)
				return a
			},
		},
		{
			name: "circle 2",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(b)
				b.Add(c)
				c.Add(a)
				return a
			},
		},
		{
			name: "add self",
			itemFunc: func(a, b, c *Item) *Item {
				a.Add(a)
				return a
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			rs := NewBackendItem("rs")
			as := NewBackendItem("as")
			vault := NewBackendItem("vault")

			defer func() { _ = recover() }()

			tt.itemFunc(rs, as, vault)

			st.Error("Missing panic")
		})
	}
}
