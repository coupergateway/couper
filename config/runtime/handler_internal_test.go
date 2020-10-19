package runtime

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/avenga/couper/config/request"
)

func TestHandler_CreatePattern(t *testing.T) {
	type testCase struct {
		input      string
		expPattern string
		expParams  []*request.PathParam
		expErr     string
	}

	for i, tc := range []testCase{
		{"a", "", nil, ""},
		{"/", "/", nil, ""},
		{"/{", "", nil, ""},
		{"/}", "", nil, ""},
		{"/{}", "", nil, ""},
		{"/abc", "/abc", nil, ""},
		{"/abc/{x}", "/abc/{}", []*request.PathParam{{Name: "x", Position: 2}}, ""},
		{"/abc/{x}/", "/abc/{}/", []*request.PathParam{{Name: "x", Position: 2}}, ""},
		{"/abc/{x}/{y}/", "/abc/{}/{}/", []*request.PathParam{{Name: "x", Position: 2}, {Name: "y", Position: 3}}, ""},
	} {
		gotPattern, gotParams, _ := createPattern(tc.input)

		if tc.expPattern != gotPattern {
			t.Errorf("%d: Expected pattern %q, got %q", i, tc.expPattern, gotPattern)
		}

		if !reflect.DeepEqual(tc.expParams, gotParams) {
			t.Errorf("%d: Expected params %q, got %q", i, fmt.Sprintf("%#v", tc.expParams), fmt.Sprintf("%#v", gotParams))
		}
	}
}
