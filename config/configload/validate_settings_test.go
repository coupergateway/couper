package configload

import (
	"strings"
	"testing"
)

// Boundaries the OTel SDK would silently replace with its millisecond defaults must not
// pass config load.
func TestSettingsMetricsRequestDurationBuckets(t *testing.T) {
	for _, tt := range []struct {
		name    string
		setting string
		error   string
	}{
		{"absent", "", ""},
		{"empty list", `beta_metrics_request_duration_buckets = []`, ""},
		{"single boundary", `beta_metrics_request_duration_buckets = [0.5]`, ""},
		{"zero boundary", `beta_metrics_request_duration_buckets = [0, 0.5]`, ""},
		{"unsorted", `beta_metrics_request_duration_buckets = [1, 0.05, 10, 0.2]`, ""},
		{
			"duplicate",
			`beta_metrics_request_duration_buckets = [0.1, 0.1, 1]`,
			"beta_metrics_request_duration_buckets must not contain the duplicate boundary 0.1",
		},
		{
			"negative",
			`beta_metrics_request_duration_buckets = [-1, 0.5]`,
			"beta_metrics_request_duration_buckets must not contain the negative boundary -1",
		},
	} {
		t.Run(tt.name, func(subT *testing.T) {
			hcl := "server {}\nsettings {\n  beta_metrics = true\n  " + tt.setting + "\n}"

			_, err := LoadBytes([]byte(hcl), "couper.hcl")

			if tt.error == "" {
				if err != nil {
					subT.Fatalf("want no error, got: %v", err)
				}
				return
			}

			if err == nil {
				subT.Fatalf("want an error containing %q, got none", tt.error)
			}
			if !strings.Contains(err.Error(), tt.error) {
				subT.Errorf("want an error containing %q, got: %v", tt.error, err)
			}
			// the diagnostic points at the attribute
			if !strings.Contains(err.Error(), "couper.hcl:4") {
				subT.Errorf("want the error to name the attribute's line, got: %v", err)
			}
		})
	}
}
