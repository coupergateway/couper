package accesscontrol_test

import (
	"testing"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/internal/test"
)

func Test_JWKS(t *testing.T) {
	helper := test.New(t)
	tests := []struct {
		name      string
		file      string
		expParsed bool
	}{
		{"RSA", "testdata/jwks.json", true},
		{"oct", "testdata/jwks_oct.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			jwks, err := ac.NewJWKS("file:"+tt.file, "", nil, nil)
			helper.Must(err)
			err = jwks.Load()
			if err != nil && tt.expParsed {
				subT.Error("no jwks parsed")
			}
			if err == nil && !tt.expParsed {
				subT.Error("no jwks expected")
			}
		})
	}
}

func Test_JWKS_GetKey(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		kid      string
		alg      string
		use      string
		expFound bool
	}{
		{"key for kid1", "testdata/jwks.json", "kid1", "RS256", "sig", true},
		{"key for kid2 sig", "testdata/jwks.json", "kid2", "RS256", "sig", true},
		{"key for key2 enc", "testdata/jwks.json", "kid2", "RS256", "enc", true},
		{"no key for kid1 enc", "testdata/jwks.json", "kid1", "RS256", "enc", false},
		{"no key for kid1 RS384", "testdata/jwks.json", "kid1", "RS384", "sig", false},
		{"no key for kid3", "testdata/jwks.json", "kid3", "RS256", "sig", false},
		{"no key for empty kid", "testdata/jwks.json", "", "RS256", "sig", false}, // or better nil kid
		{"no key for empty alg", "testdata/jwks.json", "kid1", "", "sig", false},
		{"no_use: key for sig", "testdata/jwks_no_use.json", "kid1", "RS256", "sig", false},
		{"no_use: key for empty use", "testdata/jwks_no_use.json", "kid1", "RS256", "", true}, // not useful: we always call with "sig"
		{"no_kid: key for empty kid", "testdata/jwks_no_kid.json", "", "RS256", "sig", true},  // or better nil kid?
		{"no_kid: no key for kid", "testdata/jwks_no_kid.json", "kid1", "RS256", "sig", false},
		{"no_alg: no key for empty alg", "testdata/jwks_no_alg.json", "kid1", "", "sig", true}, // or better nil alg?
		{"no_alg: no key for alg", "testdata/jwks_no_alg.json", "kid1", "RS256", "sig", false},
		{"no_kid_no_alg: key for empty kid, empty alg, sig", "testdata/jwks_no_kid_no_alg.json", "", "", "sig", true},
		{"no_kid_no_alg: key for kid, alg, sig", "testdata/jwks_no_kid_no_alg.json", "kid", "RS256", "sig", false},
		{"no_kid_no_alg_no_use: key for empty kid, empty alg, sig", "testdata/jwks_no_kid_no_alg_no_use.json", "", "", "sig", false},
		{"no_kid_no_alg_no_use: key for kid, alg, sig", "testdata/jwks_no_kid_no_alg_no_use.json", "kid", "RS256", "sig", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(t)
			jwks, err := ac.NewJWKS("file:"+tt.file, "", nil, nil)
			helper.Must(err)
			err = jwks.Load()
			helper.Must(err)
			jwk, err := jwks.GetKey(tt.kid, tt.alg, tt.use)
			if jwk == nil && tt.expFound {
				subT.Errorf("no jwk found, %v", err)
			}
			if jwk != nil && !tt.expFound {
				subT.Error("no jwk expected")
			}
		})
	}
}
