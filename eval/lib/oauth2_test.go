package lib_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/oauth2/oidc"
)

func TestNewOAuthAuthorizationURLFunction(t *testing.T) {
	helper := test.New(t)

	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var conf interface{}
		if req.URL.Path == "/.well-known/openid-configuration" {
			conf = &oidc.OpenidConfiguration{
				AuthorizationEndpoint:         origin.URL + "/auth",
				CodeChallengeMethodsSupported: []string{config.CcmS256},
				Issuer:                        "thatsme",
				JwksURI:                       origin.URL + "/jwks",
				TokenEndpoint:                 origin.URL + "/token",
				UserinfoEndpoint:              origin.URL + "/userinfo",
			}
		} else if req.URL.Path == "/jwks" {
			conf = jwk.JWKSData{}
		}

		b, err := json.Marshal(conf)
		helper.Must(err)
		_, err = rw.Write(b)
		helper.Must(err)
	}))
	defer origin.Close()

	confURL := origin.URL + "/.well-known/openid-configuration"
	authURL := origin.URL + "/auth"
	tokenURL := origin.URL + "/token"

	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())

	type testCase struct {
		name      string
		config    string
		wantRedir string
		wantScope string
		wantNonce bool
		wantState bool
		wantPKCE  bool
	}

	for _, tc := range []testCase{
		{
			"oidc",
			`server {}
definitions {
  oidc "auth-ref" {
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    configuration_url = "` + confURL + `"
    redirect_uri = "/cb"
    verifier_value = "asdf"
  }
}
`,
			"https://couper.io/cb",
			"openid",
			true,
			false,
			false,
		},
		{
			"oidc, redir URI from env var and using function",
			`server {}
definitions {
  oidc "auth-ref" {
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    configuration_url = "` + confURL + `"
    redirect_uri = split(" ", env.REDIR_URIS)[0]
    verifier_value = "asdf"
  }
}
defaults {
  environment_variables = {
    REDIR_URIS = "/cb /cb2"
  }
}
`,
			"https://couper.io/cb",
			"openid",
			true,
			false,
			false,
		},
		{
			"oidc, absolute redir URI",
			`server {}
definitions {
  oidc "auth-ref" {
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    configuration_url = "` + confURL + `"
    redirect_uri = "https://example.com/cb"
    verifier_value = "asdf"
  }
}
`,
			"https://example.com/cb",
			"openid",
			true,
			false,
			false,
		},
		{
			"oidc, empty scope",
			`server {}
definitions {
  oidc "auth-ref" {
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    configuration_url = "` + confURL + `"
    redirect_uri = "/cb"
    verifier_value = "asdf"
    scope = ""
  }
}
`,
			"https://couper.io/cb",
			"openid",
			true,
			false,
			false,
		},
		{
			"oidc, email scope",
			`server {}
definitions {
  oidc "auth-ref" {
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    configuration_url = "` + confURL + `"
    redirect_uri = "/cb"
    verifier_value = "asdf"
    scope = "email"
  }
}
`,
			"https://couper.io/cb",
			"openid email",
			true,
			false,
			false,
		},
		{
			"oidc, verifier_method ccm_s256",
			`server {}
definitions {
  oidc "auth-ref" {
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    configuration_url = "` + confURL + `"
    redirect_uri = "/cb"
    verifier_method = "ccm_s256"
    verifier_value = "asdf"
  }
}
`,
			"https://couper.io/cb",
			"openid",
			false,
			false,
			true,
		},
		{
			"oauth2 authorization code",
			`server {}
definitions {
  beta_oauth2 "auth-ref" {
    grant_type = "authorization_code"
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    authorization_endpoint = "` + authURL + `"
    token_endpoint = "` + tokenURL + `"
    redirect_uri = "/cb"
    verifier_method = "state"
    verifier_value = "asdf"
  }
}
`,
			"https://couper.io/cb",
			"",
			false,
			true,
			false,
		},
		{
			"oauth2 authorization code, redir URI from env var",
			`server {}
definitions {
  beta_oauth2 "auth-ref" {
    grant_type = "authorization_code"
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    authorization_endpoint = "` + authURL + `"
    token_endpoint = "` + tokenURL + `"
    redirect_uri = env.REDIR_URI
    verifier_method = "state"
    verifier_value = "asdf"
  }
}
defaults {
  environment_variables = {
    REDIR_URI = "/cb"
  }
}
`,
			"https://couper.io/cb",
			"",
			false,
			true,
			false,
		},
		{
			"oauth2 authorization code, absolute redir URI",
			`server {}
definitions {
  beta_oauth2 "auth-ref" {
    grant_type = "authorization_code"
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    authorization_endpoint = "` + authURL + `"
    token_endpoint = "` + tokenURL + `"
    redirect_uri = "https://example.com/cb"
    verifier_method = "state"
    verifier_value = "asdf"
  }
}
`,
			"https://example.com/cb",
			"",
			false,
			true,
			false,
		},
		{
			"oauth2 authorization code, empty scope",
			`server {}
definitions {
  beta_oauth2 "auth-ref" {
    grant_type = "authorization_code"
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    authorization_endpoint = "` + authURL + `"
    token_endpoint = "` + tokenURL + `"
    redirect_uri = "/cb"
    verifier_method = "state"
    verifier_value = "asdf"
    scope = ""
  }
}
`,
			"https://couper.io/cb",
			"",
			false,
			true,
			false,
		},
		{
			"oauth2 authorization code, 'foo bar' scope",
			`server {}
definitions {
  beta_oauth2 "auth-ref" {
    grant_type = "authorization_code"
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    authorization_endpoint = "` + authURL + `"
    token_endpoint = "` + tokenURL + `"
    redirect_uri = "/cb"
    verifier_method = "state"
    verifier_value = "asdf"
    scope = "foo bar"
  }
}
`,
			"https://couper.io/cb",
			"foo bar",
			false,
			true,
			false,
		},
		{
			"oauth2 authorization code, verifier_method ccm_s256",
			`server {}
definitions {
  beta_oauth2 "auth-ref" {
    grant_type = "authorization_code"
    client_id = "test-id"
    client_secret = "test-s3cr3t"
    authorization_endpoint = "` + authURL + `"
    token_endpoint = "` + tokenURL + `"
    redirect_uri = "/cb"
    verifier_method = "ccm_s256"
    verifier_value = "asdf"
  }
}
`,
			"https://couper.io/cb",
			"",
			false,
			false,
			true,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			couperConf, err := configload.LoadBytes([]byte(tc.config), "test.hcl")
			h.Must(err)

			quitCh := make(chan struct{}, 1)
			defer close(quitCh)
			memStore := cache.New(logger, quitCh)

			ctx, cancel := context.WithCancel(couperConf.Context)
			couperConf.Context = ctx
			defer cancel()

			_, err = runtime.NewServerConfiguration(couperConf, logger, memStore)
			helper.Must(err)

			req, rerr := http.NewRequest(http.MethodGet, "https://couper.io/", nil)
			helper.Must(rerr)
			req = req.Clone(context.Background())

			hclCtx := couperConf.Context.(*eval.Context).
				WithClientRequest(req).
				HCLContext()

			val, furr := hclCtx.Functions[lib.FnOAuthAuthorizationURL].Call([]cty.Value{cty.StringVal("auth-ref")})
			helper.Must(furr)

			authURL := seetie.ValueToString(val)
			authURLObj, perr := url.Parse(authURL)
			helper.Must(perr)

			query := authURLObj.Query()

			if rt := query.Get("response_type"); rt != "code" {
				subT.Errorf("response_type want: %v; got: %v", "code", rt)
			}

			if clID := query.Get("client_id"); clID != "test-id" {
				subT.Errorf("client_id want: %v; got: %v", "test-id", clID)
			}

			if redir := query.Get("redirect_uri"); redir != tc.wantRedir {
				subT.Errorf("redirect_uri want: %v; got: %v", tc.wantRedir, redir)
			}

			if tc.wantScope == "" && query.Has("scope") {
				subT.Error("scope not expected")
			}
			if scope := query.Get("scope"); scope != tc.wantScope {
				subT.Errorf("scope want: %v; got: %v", tc.wantScope, scope)
			}

			nonce := query.Get("nonce")
			if tc.wantNonce {
				if nonce == "" {
					subT.Error("missing nonce")
				}
			} else {
				if nonce != "" {
					subT.Error("nonce not expected")
				}
			}

			state := query.Get("state")
			if tc.wantState {
				if state == "" {
					subT.Error("missing state")
				}
			} else {
				if state != "" {
					subT.Error("state not expected")
				}
			}

			codeChallenge := query.Get("code_challenge")
			codeChallengeMethod := query.Get("code_challenge_method")
			if tc.wantPKCE {
				if codeChallenge == "" {
					subT.Error("missing code_challenge")
				}
				if codeChallengeMethod != "S256" {
					subT.Errorf("code_challenge want: %v; got: %v\n", "S256", codeChallengeMethod)
				}
			} else {
				if codeChallenge != "" {
					subT.Error("code_challenge not expected")
				}
			}
		})
	}
}

func TestOAuthAuthorizationURLError(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		label    string
		wantErr  string
	}{
		{
			"missing oidc/beta_oauth2 definitions",
			`
			server {}
			definitions {
			}
			`,
			"MyLabel",
			`missing oidc or beta_oauth2 block with referenced label "MyLabel"`,
		},
		{
			"missing referenced oidc/beta_oauth2",
			`
			server {}
			definitions {
			  beta_oauth2 "auth-ref" {
			    grant_type = "authorization_code"
			    client_id = "test-id"
			    client_secret = "test-s3cr3t"
			    authorization_endpoint = "https://a.s./auth"
			    token_endpoint = "https://a.s./token"
			    redirect_uri = "/cb"
			    verifier_method = "ccm_s256"
			    verifier_value = "asdf"
			  }
			}
			`,
			"MyLabel",
			`missing oidc or beta_oauth2 block with referenced label "MyLabel"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := test.New(subT)
			couperConf, err := configload.LoadBytes([]byte(tt.config), "test.hcl")
			h.Must(err)

			ctx, cancel := context.WithCancel(couperConf.Context)
			couperConf.Context = ctx
			defer cancel()

			evalContext := couperConf.Context.Value(request.ContextType).(*eval.Context)
			req, err := http.NewRequest(http.MethodGet, "https://www.example.com/foo", nil)
			h.Must(err)
			evalContext = evalContext.WithClientRequest(req)

			_, err = evalContext.HCLContext().Functions[lib.FnOAuthAuthorizationURL].Call([]cty.Value{cty.StringVal(tt.label)})
			if err == nil {
				subT.Error("expected an error, got nothing")
				return
			}
			if err.Error() != tt.wantErr {
				subT.Errorf("\nWant:\t%q\nGot:\t%q", tt.wantErr, err.Error())
			}
		})
	}
}
