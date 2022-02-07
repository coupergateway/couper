package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func TestAccessControl_ErrorHandler(t *testing.T) {
	client := newClient()

	shutdown, logHook := newCouper("testdata/integration/error_handler/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name          string
		header        test.Header
		expLogMsg     string
		expStatusCode int
	}

	for _, tc := range []testCase{
		{"catch all", test.Header{"Authorization": "Basic aGFuczpoYW5z"}, "access control error: ba: credential mismatch", http.StatusNotFound},
		{"catch specific", nil, "access control error: ba: credentials required", http.StatusBadGateway},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
			helper.Must(err)

			tc.header.Set(req)

			res, err := client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != tc.expStatusCode {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.name, tc.expStatusCode, res.StatusCode)
			}

			if logHook.LastEntry().Data["status"] != tc.expStatusCode {
				subT.Logf("%v", logHook.LastEntry())
				subT.Errorf("Expected statusCode log: %d", tc.expStatusCode)
			}

			if logHook.LastEntry().Message != tc.expLogMsg {
				subT.Logf("%v", logHook.LastEntry())
				subT.Errorf("Expected message log: %s", tc.expLogMsg)
			}
		})
	}
}

func TestAccessControl_ErrorHandler_BasicAuth_Default(t *testing.T) {
	client := test.NewHTTPClient()

	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/error_handler/01_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/default/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected Status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
		return
	}

	if www := res.Header.Get("www-authenticate"); www != "Basic realm=protected" {
		t.Errorf("Expected header: www-authenticate with value: %s, got: %s", "Basic realm=protected", www)
	}
}

func TestAccessControl_ErrorHandler_BasicAuth_Wildcard(t *testing.T) {
	client := test.NewHTTPClient()

	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/error_handler/02_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected Status %d, got: %d", http.StatusOK, res.StatusCode)
		return
	}

	if www := res.Header.Get("www-authenticate"); www != "" {
		t.Errorf("Expected no www-authenticate header: %s", www)
	}
}

func TestAccessControl_ErrorHandler_Configuration_Error(t *testing.T) {
	_, err := configload.LoadFiles("testdata/integration/error_handler/03_couper.hcl", "")

	expectedMsg := "03_couper.hcl:24,12-12: Missing required argument; The argument \"grant_type\" is required, but no definition was found."

	if err == nil {
		t.Error("config error should not be nil")
	} else if !strings.HasSuffix(err.Error(), expectedMsg) {
		t.Errorf("\nwant:\t%s\ngot:\t%v", expectedMsg, err.Error())
	}
}

func TestAccessControl_ErrorHandler_Scopes(t *testing.T) {
	client := test.NewHTTPClient()

	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/error_handler/04_couper.hcl", helper)
	defer shutdown()

	type testcase struct {
		Name      string
		Method    string
		Path      string
		Scopes    []string
		ExpStatus int
		ExpBody   string
	}

	for _, tc := range []testcase{
		{"api: w/ scope", http.MethodGet, "/api/", []string{"read"}, http.StatusNoContent, ""},
		{"api: w/ wrong scope; handle scope", http.MethodGet, "/api/", []string{"another"}, http.StatusTeapot, ""},
		{"api pow: w/ scope; op granted", http.MethodPost, "/api/pow/", []string{"read", "power"}, http.StatusNoContent, ""},
		{"api pow: w/ wrong scope; handle insufficient_scope", http.MethodPost, "/api/pow/", []string{"read", "another"}, http.StatusBadRequest, ""},
		{"api pow: w/ scope method; handle operation_denied", http.MethodGet, "/api/pow/", []string{"read", "another"}, http.StatusMethodNotAllowed, ""},
		{"endpoint: w/ scope", http.MethodGet, "/", []string{"write"}, http.StatusOK, ""},
		{"endpoint: w/ wrong scope; handle scope", http.MethodGet, "/", []string{"another"}, http.StatusTeapot, ""},
	} {
		t.Run(tc.Name, func(st *testing.T) {
			h := test.New(st)
			req, err := http.NewRequest(tc.Method, "http://localhost:8080"+tc.Path, nil)
			h.Must(err)

			conf, err := lib.NewJWTSigningConfigFromJWT(&config.JWT{
				Name:               "test",
				Key:                "s3cr3t", // same as config file
				SignatureAlgorithm: jwt.AlgorithmHMAC256.String(),
				SigningTTL:         "5m", // required for jwt sign func
			})
			h.Must(err)

			ctx := eval.ContextFromRequest(req)
			ctx = ctx.WithJWTSigningConfigs(map[string]*lib.JWTSigningConfig{
				"test": conf,
			})
			ctx = ctx.WithClientRequest(req)

			tokenVal, err := ctx.HCLContext().Functions[lib.FnJWTSign].
				Call([]cty.Value{cty.StringVal("test"), seetie.MapToValue(
					map[string]interface{}{
						"scopes": tc.Scopes,
					})})
			h.Must(err)

			req.Header.Set("Authorization", "Bearer "+seetie.ValueToString(tokenVal))

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.ExpStatus {
				st.Errorf("Expected statusCode: %d, got: %d", tc.ExpStatus, res.StatusCode)
			}
		})
	}

}
