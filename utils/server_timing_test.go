package utils_test

import (
	"testing"

	"github.com/coupergateway/couper/utils"
	"github.com/google/go-cmp/cmp"
)

func TestUtils_CollectMetricNames(t *testing.T) {
	type testCase struct {
		header string
		exp    utils.ServerTimings
	}

	for _, tc := range []testCase{
		{
			`miss`, utils.ServerTimings{`miss`: ``},
		},
		{
			`miss, db;dur=1`, utils.ServerTimings{`miss`: ``, `db`: ``},
		},
		{
			`=`, utils.ServerTimings{},
		},
		{
			`;, X`, utils.ServerTimings{`X`: ``},
		},
		{
			"X" + string([]byte{4}), utils.ServerTimings{`X`: ``},
		},
		{
			`miss;desc="...", X;DB=1`, utils.ServerTimings{`miss`: ``, `X`: ``},
		},
		{
			`miss;desc=".,.", X;DB=1`, utils.ServerTimings{`miss`: ``, `X`: ``},
		},
		{
			`miss;desc="`, utils.ServerTimings{`miss`: ``},
		},
	} {
		serverTimings := make(utils.ServerTimings)

		utils.CollectMetricNames(tc.header, serverTimings)

		if !cmp.Equal(tc.exp, serverTimings) {
			t.Errorf("%s", cmp.Diff(tc.exp, serverTimings))
		}
	}
}

func TestUtils_MergeMetrics(t *testing.T) {
	type testCase struct {
		src  utils.ServerTimings
		dest utils.ServerTimings
		exp  int
	}

	for _, tc := range []testCase{
		{
			utils.ServerTimings{`db`: ``},
			utils.ServerTimings{`db`: ``},
			2,
		},
	} {
		utils.MergeMetrics(tc.src, tc.dest)

		if tc.exp != len(tc.dest) {
			t.Errorf("%#v", tc.dest)
		}

		var newSrc = make(utils.ServerTimings)
		for k, v := range tc.dest {
			newSrc[k] = v
		}

		utils.MergeMetrics(newSrc, tc.dest)

		if tc.exp != len(tc.dest)/2 {
			t.Errorf("%#v", tc.dest)
		}
	}
}
