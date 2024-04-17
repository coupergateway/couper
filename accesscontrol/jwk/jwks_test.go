package jwk_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/coupergateway/couper/accesscontrol/jwk"
	"github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/handler/transport"
	"github.com/coupergateway/couper/internal/test"
)

func Test_JWKS(t *testing.T) {

	origin := test.NewBackend()
	defer origin.Close()

	log, _ := test.NewLogger()

	backend := transport.NewBackend(body.NewHCLSyntaxBodyWithStringAttr("origin", origin.Addr()),
		&transport.Config{}, nil, log.WithContext(context.Background()))

	tests := []struct {
		name  string
		url   string
		error string
	}{
		{"missing_scheme", "no-scheme", `unsupported JWKS URI scheme: "no-scheme"`},
		{"short url", "file", `unsupported JWKS URI scheme: "file"`},
		{"short url", "https", `unsupported JWKS URI scheme: "https"`},
		{"ok file", "file:testdata/jwks.json", ""},
		{"ok http", origin.Addr() + "/jwks.json", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := jwk.NewJWKS(context.TODO(), tt.url, "", "", backend)
			if err == nil && tt.error != "" {
				subT.Errorf("Missing error:\n\tWant: %v\n\tGot:  %v", tt.error, nil)
			}
			if err != nil && err.Error() != tt.error {
				subT.Errorf("Unexpected error:\n\tWant: %v\n\tGot:  %#v", tt.error, err)
			}
		})
	}
}

func Test_JWKS_Load(t *testing.T) {
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
			jwks, err := jwk.NewJWKS(context.TODO(), "file:"+tt.file, "", "", nil)
			helper.Must(err)
			_, err = jwks.Data()
			if err != nil && tt.expParsed {
				subT.Error("no jwks parsed")
			}
			if err == nil && !tt.expParsed {
				subT.Error("no jwks expected")
			}
		})
	}
}

func Test_JWKS_GetSigKeyForToken(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		kid      interface{}
		alg      interface{}
		expFound bool
	}{
		{"non-empty kid, non-empty alg", "testdata/jwks.json", "kid1", "RS256", true},
		{"nil kid, non-empty alg", "testdata/jwks_no_kid.json", nil, "RS256", true},
		{"non-empty kid, nil alg", "testdata/jwks_no_alg.json", "kid1", nil, false},
		{"nil kid, nil alg", "testdata/jwks_no_kid_no_alg.json", nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(subT)
			jwks, err := jwk.NewJWKS(context.TODO(), "file:"+tt.file, "", "", nil)
			helper.Must(err)
			_, err = jwks.Data()
			helper.Must(err)
			token := &jwt.Token{Header: map[string]interface{}{"kid": tt.kid, "alg": tt.alg}}
			jwk, err := jwks.GetSigKeyForToken(token)
			if jwk == nil && tt.expFound {
				subT.Errorf("no jwk found, %v", err)
			}
			if jwk != nil && !tt.expFound {
				subT.Error("no jwk expected")
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
		{"no_alg: key for empty alg", "testdata/jwks_no_alg.json", "kid1", "", "sig", true}, // or better nil alg?
		{"no_alg: key for alg", "testdata/jwks_no_alg.json", "kid1", "RS256", "sig", true},
		{"no_kid_no_alg: key for empty kid, empty alg, sig", "testdata/jwks_no_kid_no_alg.json", "", "", "sig", true},
		{"no_kid_no_alg: key for kid, alg, sig", "testdata/jwks_no_kid_no_alg.json", "kid", "RS256", "sig", false},
		{"no_kid_no_alg_no_use: key for empty kid, empty alg, sig", "testdata/jwks_no_kid_no_alg_no_use.json", "", "", "sig", false},
		{"no_kid_no_alg_no_use: key for kid, alg, sig", "testdata/jwks_no_kid_no_alg_no_use.json", "kid", "RS256", "sig", false},
		{"missing crv", "testdata/jwks_ecdsa.json", "missing-crv", "ES256", "sig", false},
		{"invalid crv", "testdata/jwks_ecdsa.json", "invalid-crv", "ES512", "sig", false},
		{"missing x", "testdata/jwks_ecdsa.json", "missing-x", "ES256", "sig", false},
		{"missing y", "testdata/jwks_ecdsa.json", "missing-y", "ES256", "sig", false},
		{"ok", "testdata/jwks_ecdsa.json", "ok", "ES256", "sig", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(subT)
			jwks, err := jwk.NewJWKS(context.TODO(), "file:"+tt.file, "", "", nil)
			helper.Must(err)
			_, err = jwks.Data()
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

func Test_JWKS_LoadSynced(t *testing.T) {
	helper := test.New(t)

	memQuitCh := make(chan struct{})
	defer close(memQuitCh)

	jwksOrigin := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		f, err := os.ReadFile("testdata/jwks.json")
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.Copy(writer, bytes.NewReader(f))
	}))
	defer jwksOrigin.Close()

	jwks, err := jwk.NewJWKS(context.TODO(), jwksOrigin.URL, "10s", "", http.DefaultTransport)
	helper.Must(err)

	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer wg.Done()
			_, e := jwks.GetKey("kid1", "", "")
			helper.Must(e)
		}(i)
	}
	wg.Wait()
}
