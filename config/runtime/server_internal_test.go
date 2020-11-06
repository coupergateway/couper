package runtime

import (
	"testing"

	"github.com/avenga/couper/config"
)

func TestServer_isUnique(t *testing.T) {
	endpoints := make(map[string]bool)

	type testCase struct {
		pattern string
		clean   string
		unique  bool
	}

	for i, tc := range []testCase{
		{"/abc", "/abc", true},
		{"/abc", "/abc", false},
		{"/abc/", "/abc/", true},
		{"/x/{xxx}", "/x/{}", true},
		{"/x/{yyy}", "/x/{}", false},
		{"/x/{xxx}/a/{yyy}", "/x/{}/a/{}", true},
		{"/x/{yyy}/a/{xxx}", "/x/{}/a/{}", false},
	} {
		unique, cleanPattern := isUnique(endpoints, tc.pattern)
		endpoints[cleanPattern] = true

		if unique != tc.unique {
			t.Errorf("%d: Unexpected unique status given: %t", i+1, unique)
		}
		if cleanPattern != tc.clean {
			t.Errorf("%d: Unexpected cleanPattern given: %s, want %s", i+1, cleanPattern, tc.clean)
		}
	}
}

func TestServer_validatePortHosts(t *testing.T) {
	t.Skip("TODO")
	type args struct {
		conf           *config.Gateway
		configuredPort int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePortHosts(tt.args.conf, tt.args.configuredPort); (err != nil) != tt.wantErr {
				t.Errorf("validatePortHosts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
