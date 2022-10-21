package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/oauth2"
)

func TestEndpoints_OAuth2(t *testing.T) {
	helper := test.New(t)

	for i := range []int{0, 1, 2} {
		var seenCh, tokenSeenCh chan struct{}

		retries := 0

		oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/oauth2" {
				if accept := req.Header.Get("Accept"); accept != "application/json" {
					t.Errorf("expected Accept %q, got: %q", "application/json", accept)
				}

				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)

				body := []byte(`{
					"access_token": "abcdef0123456789",
					"token_type": "bearer",
					"expires_in": 100
				}`)
				_, werr := rw.Write(body)
				helper.Must(werr)

				// retries must be equal with the number of retries in the `testdata/oauth2/XXX_retries_couper.hcl`
				if retries == i {
					close(tokenSeenCh)
				}

				return
			}
			rw.WriteHeader(http.StatusBadRequest)
		}))
		defer oauthOrigin.Close()

		ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/resource" {
				// retries must be equal with the number of retries in the `testdata/oauth2/XXX_retries_couper.hcl`
				if req.Header.Get("Authorization") == "Bearer abcdef0123456789" && retries == i {
					rw.WriteHeader(http.StatusNoContent)
					close(seenCh)
					return
				}

				retries++

				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			rw.WriteHeader(http.StatusNotFound)
		}))
		defer ResourceOrigin.Close()

		confPath := fmt.Sprintf("testdata/oauth2/%d_retries_couper.hcl", i)
		shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL, "rsOrigin": ResourceOrigin.URL})
		defer shutdown()

		req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
		helper.Must(err)

		for _, p := range []string{"/", "/2nd", "/password"} {
			hook.Reset()

			seenCh = make(chan struct{})
			tokenSeenCh = make(chan struct{})

			req.URL.Path = p
			res, err := newClient().Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusNoContent {
				t.Errorf("expected status NoContent, got: %d", res.StatusCode)
				return
			}

			timer := time.NewTimer(time.Second * 2)
			select {
			case <-timer.C:
				t.Error("OAuth2 request failed")
			case <-tokenSeenCh:
				<-seenCh
			}
		}

		oauthOrigin.Close()
		ResourceOrigin.Close()
		shutdown()
	}
}

func Test_OAuth2_no_retry(t *testing.T) {
	// tests that actually no retry is attempted for oauth2 with retries = 0
	helper := test.New(t)

	retries := 0

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth2" {
			if accept := req.Header.Get("Accept"); accept != "application/json" {
				t.Errorf("expected Accept %q, got: %q", "application/json", accept)
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/resource" {
			if retries > 0 {
				t.Fatal("Must not retry")
			}

			retries++

			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
	defer ResourceOrigin.Close()

	confPath := "testdata/oauth2/0_retries_couper.hcl"
	shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL, "rsOrigin": ResourceOrigin.URL})
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(err)

	hook.Reset()

	req.URL.Path = "/"
	res, err := newClient().Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
		return
	}

	oauthOrigin.Close()
	ResourceOrigin.Close()
	shutdown()
}

func TestEndpoints_OAuth2_Options(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		configFile string
		expBody    string
		expAuth    string
	}

	for _, tc := range []testCase{
		{
			"01_couper.hcl",
			`client_id=user&client_secret=pass+word&grant_type=client_credentials&scope=scope1+scope2`,
			"",
		},
		{
			"02_couper.hcl",
			`grant_type=client_credentials`,
			"Basic dXNlcjpwYXNzJTJCK3dvcmQ=",
		},
		{
			"03_couper.hcl",
			`grant_type=client_credentials`,
			"Basic dXNlcjpwYXNz",
		},
		{
			"12_couper.hcl",
			`grant_type=password&password=pass&scope=scope1+scope2&username=user`,
			"Basic bXlfY2xpZW50Om15X2NsaWVudF9zZWNyZXQ=",
		},
		{
			"13_couper.hcl",
			`client_id=my_client&client_secret=my_client_secret&grant_type=password&password=pass&scope=scope1+scope2&username=user`,
			"",
		},
		{
			"16_couper.hcl",
			`assertion=GET&grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer`,
			"",
		},
	} {
		var tokenSeenCh chan struct{}

		oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/options" {
				reqBody, _ := io.ReadAll(req.Body)
				authorization := req.Header.Get("Authorization")

				if tc.expBody != string(reqBody) {
					t.Errorf("want\n%s\ngot\n%s", tc.expBody, reqBody)
				}
				if tc.expAuth != authorization {
					t.Errorf("want\n%s\ngot\n%s", tc.expAuth, authorization)
				}

				rw.WriteHeader(http.StatusNoContent)

				close(tokenSeenCh)
				return
			}
			rw.WriteHeader(http.StatusBadRequest)
		}))
		defer oauthOrigin.Close()

		confPath := fmt.Sprintf("testdata/oauth2/%s", tc.configFile)
		shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
		defer shutdown()

		req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
		helper.Must(err)

		hook.Reset()

		tokenSeenCh = make(chan struct{})

		req.URL.Path = "/"
		_, err = newClient().Do(req)
		helper.Must(err)

		timer := time.NewTimer(time.Second * 2)
		select {
		case <-timer.C:
			t.Error("OAuth2 request failed")
		case <-tokenSeenCh:
		}

		oauthOrigin.Close()
		shutdown()
	}
}

func TestOAuth2_Config_Errors(t *testing.T) {
	log, _ := test.NewLogger()

	type testCase struct {
		name  string
		hcl   string
		error string
	}

	for _, tc := range []testCase{
		{
			"grant_type client_credentials without client_id",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
    }
  }
}
`,
			"configuration error: be: client_id must not be empty",
		},
		{
			"grant_type password without client_id",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_secret  = "my_client_secret"
      grant_type     = "password"
      username       = "my_user"
      password       = "my_password"
    }
  }
}
`,
			"configuration error: be: client_id must not be empty",
		},
		{
			"username with grant_type client_credentials",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      username       = "my_user"
    }
  }
}
`,
			"configuration error: be: username must not be set with grant_type=client_credentials",
		},
		{
			"password with grant_type client_credentials",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      password       = "my_password"
    }
  }
}
`,
			"configuration error: be: password must not be set with grant_type=client_credentials",
		},
		{
			"username with grant_type jwt-bearer",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      grant_type     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
      username       = "my_user"
    }
  }
}
`,
			"configuration error: be: username must not be set with grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer",
		},
		{
			"password with grant_type jwt-bearer",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      grant_type     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
      password       = "my_password"
    }
  }
}
`,
			"configuration error: be: password must not be set with grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer",
		},
		{
			"assertion with grant_type client_credentials",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      assertion      = "my_assertion"
    }
  }
}
`,
			"configuration error: be: assertion must not be set with grant_type=client_credentials",
		},
		{
			"assertion with grant_type password",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "password"
      username       = "my_user"
      password       = "my_password"
      assertion      = "my_assertion"
    }
  }
}
`,
			"configuration error: be: assertion must not be set with grant_type=password",
		},
		{
			"missing username with grant_type password",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "password"
    }
  }
}
`,
			"configuration error: be: username must not be empty with grant_type=password",
		},
		{
			"missing password with grant_type password",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "password"
      username       = "my_user"
    }
  }
}
`,
			"configuration error: be: password must not be empty with grant_type=password",
		},
		{
			"missing assertion with grant_type jwt-bearer",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      grant_type     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
    }
  }
}
`,
			"configuration error: be: missing assertion with grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer",
		},

		{
			"unsupported token_endpoint_auth_method",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "unknown"
    }
  }
}
`,
			`configuration error: be: token_endpoint_auth_method "unknown" not supported`,
		},

		{
			"missing client_secret with client_secret_basic",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
    }
  }
}
`,
			"configuration error: be: client_secret must not be empty with client_secret_basic",
		},
		{
			"missing client_secret with client_secret_post",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_post"
    }
  }
}
`,
			"configuration error: be: client_secret must not be empty with client_secret_post",
		},
		{
			"missing client_secret with client_secret_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_jwt"
    }
  }
}
`,
			"configuration error: be: client_secret must not be empty with client_secret_jwt",
		},
		{
			"client_secret with private_key_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "private_key_jwt"
    }
  }
}
`,
			"configuration error: be: client_secret must not be set with private_key_jwt",
		},

		{
			"jwt_signing_profile with client_secret_basic",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_basic"
      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: jwt_signing_profile block must not be set with client_secret_basic",
		},
		{
			"jwt_signing_profile with client_secret_post",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_post"
      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: jwt_signing_profile block must not be set with client_secret_post",
		},
		{
			"inappropriate authn algorithm with client_secret_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_jwt"
      jwt_signing_profile {
        signature_algorithm = "RS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: inappropriate signature algorithm with client_secret_jwt",
		},
		{
			"inappropriate authn algorithm with private_key_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "private_key_jwt"
      authn_key = "a key"
      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: inappropriate signature algorithm with private_key_jwt",
		},

		{
			"invalid authn ttl with client_secret_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_jwt"
      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10"
      }
    }
  }
}
`,
			`configuration error: be: time: missing unit in duration "10"`,
		},
		{
			"invalid authn ttl with private_key_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "private_key_jwt"
      jwt_signing_profile {
        signature_algorithm = "RS256"
        key = "a key"
        ttl = "10"
      }
    }
  }
}
`,
			`configuration error: be: time: missing unit in duration "10"`,
		},

		{
			"authn key with client_secret_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_jwt"
      jwt_signing_profile {
        key = "a key"
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: key must not be set with client_secret_jwt",
		},
		{
			"authn key value not being a valid key",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "private_key_jwt"
      jwt_signing_profile {
        signature_algorithm = "RS256"
        ttl = "10s"
        key = "not an RSA private key"
      }
    }
  }
}
`,
			"configuration error: be: invalid Key: Key must be PEM encoded PKCS1 or PKCS8 private key",
		},

		{
			"authn key_file with client_secret_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      client_secret  = "my_client_secret"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "client_secret_jwt"
      jwt_signing_profile {
        key_file = "a_key_file"
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: key_file must not be set with client_secret_jwt",
		},
		{
			"missing authn key/key_file with private_key_jwt",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "private_key_jwt"
      jwt_signing_profile {
        signature_algorithm = "RS256"
        ttl = "10s"
      }
    }
  }
}
`,
			"configuration error: be: key and key_file must not both be empty with private_key_jwt",
		},
		{
			"key_file referencing non-existing file",
			`server {}
definitions {
  backend "be" {
    oauth2 {
      token_endpoint = "https://authorization.server/token"
      client_id      = "my_client"
      grant_type     = "client_credentials"
      token_endpoint_auth_method = "private_key_jwt"
      jwt_signing_profile {
        signature_algorithm = "RS256"
        ttl = "10s"
        key_file = "unknown"
      }
    }
  }
}
`,
			"configuration error: be: client authentication key: read error: open ",
		},
	} {
		var errMsg string
		conf, err := configload.LoadBytes([]byte(tc.hcl), "couper.hcl")
		if conf != nil {
			logger := log.WithContext(context.TODO())

			tmpStoreCh := make(chan struct{})
			defer close(tmpStoreCh)

			ctx, cancel := context.WithCancel(conf.Context)
			conf.Context = ctx
			defer cancel()

			_, err = runtime.NewServerConfiguration(conf, logger, cache.New(logger, tmpStoreCh))
		}

		if err != nil {
			if _, ok := err.(errors.GoError); ok {
				errMsg = err.(errors.GoError).LogError()
			} else {
				errMsg = err.Error()
			}
		}

		if !strings.HasPrefix(errMsg, tc.error) {
			t.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tc.name, tc.error, errMsg)
		}
	}
}

func TestOAuth2_AuthnJWT(t *testing.T) {
	helper := test.New(t)
	jtiRE, err := regexp.Compile("^[a-zA-Z0-9]{43}$")
	helper.Must(err)

	rsOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		authz := req.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			helper.Must(fmt.Errorf("wrong authz: %q", authz))
		}
		token := strings.TrimPrefix(authz, "Bearer ")
		parts := strings.Split(token, " ")
		if len(parts) != 3 {
			helper.Must(fmt.Errorf("wrong token: %q", token))
		}
		exp, err := strconv.Atoi(parts[1])
		helper.Must(err)
		iat, err := strconv.Atoi(parts[0])
		helper.Must(err)
		if exp-iat != 10 {
			helper.Must(fmt.Errorf("wrong token: %q", token))
		}
		if !jtiRE.MatchString(parts[2]) {
			helper.Must(fmt.Errorf("wrong jti: %q", parts[2]))
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer rsOrigin.Close()

	type testCase struct {
		name       string
		path       string
		wantStatus int
		wantErrLog string
	}

	for _, tc := range []testCase{
		{
			"client_secret_jwt",
			"/csj",
			http.StatusNoContent,
			"",
		},
		{
			"client_secret_jwt error",
			"/csj_error",
			http.StatusBadGateway,
			"access control error: csj_error: token signature is invalid",
		},
		{
			"private_key_jwt",
			"/pkj",
			http.StatusNoContent,
			"",
		},
		{
			"private_key_jwt error",
			"/pkj_error",
			http.StatusBadGateway,
			"access control error: pkj_error: token is unverifiable: signing method RS256 is invalid",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			shutdown, hook := newCouperWithTemplate("testdata/oauth2/20_couper.hcl", h, map[string]interface{}{"rsOrigin": rsOrigin.URL})
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080"+tc.path, nil)
			h.Must(err)

			hook.Reset()

			res, err := newClient().Do(req)
			h.Must(err)

			if res.StatusCode != tc.wantStatus {
				t.Errorf("expected status %d, got: %d", tc.wantStatus, res.StatusCode)
			}

			message := getFirstAccessLogMessage(hook)
			if message != tc.wantErrLog {
				t.Errorf("error log\nwant: %q\ngot:  %q", tc.wantErrLog, message)
			}

			shutdown()
		})
	}

	rsOrigin.Close()
}

func TestOAuth2_Runtime_Errors(t *testing.T) {
	helper := test.New(t)

	asOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer asOrigin.Close()

	type testCase struct {
		name       string
		filename   string
		wantErrLog string
	}

	for _, tc := range []testCase{
		{"null assertion", "17_couper.hcl", "backend error: be: request error: oauth2: assertion expression evaluates to null"},
		{"non-string assertion", "18_couper.hcl", "backend error: be: request error: oauth2: assertion expression must evaluate to a string"},
		{"token request error", "19_couper.hcl", "backend error: be: request error: oauth2: token request failed"},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			shutdown, hook := newCouperWithTemplate("testdata/oauth2/"+tc.filename, h, map[string]interface{}{"asOrigin": asOrigin.URL})
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/resource", nil)
			h.Must(err)

			hook.Reset()

			res, err := newClient().Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusBadGateway {
				t.Errorf("expected status StatusBadGateway, got: %d", res.StatusCode)
			}

			message := getFirstAccessLogMessage(hook)
			if message != tc.wantErrLog {
				t.Errorf("error log\nwant: %q\ngot:  %q", tc.wantErrLog, message)
			}

			shutdown()
		})
	}

	asOrigin.Close()
}

func TestOAuth2_AccessControl(t *testing.T) {
	client := newClient()

	st := "qeirtbnpetrbi"
	state := oauth2.Base64urlSha256(st)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		errResp := func(err error) {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(err.Error()))
		}

		if req.URL.Path == "/token" {
			if accept := req.Header.Get("Accept"); accept != "application/json" {
				t.Errorf("expected Accept %q, got: %q", "application/json", accept)
			}
			_ = req.ParseForm()
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			code := req.PostForm.Get("code")
			idTokenToAdd := ""
			if strings.HasSuffix(code, "-id") {
				nonce := state
				mapClaims := jwt.MapClaims{}
				if !strings.HasSuffix(code, "-maud-id") {
					if strings.HasSuffix(code, "-waud-id") {
						mapClaims["aud"] = "another-client-id"
					} else if strings.HasSuffix(code, "-naud-id") {
						mapClaims["aud"] = nil
					} else {
						mapClaims["aud"] = []string{"foo", "another-client-id"}
					}
				}
				if !strings.HasSuffix(code, "-miss-id") {
					if strings.HasSuffix(code, "-wiss-id") {
						mapClaims["iss"] = "https://malicious.authorization.server"
					} else {
						mapClaims["iss"] = "https://authorization.server"
					}
				}
				if !strings.HasSuffix(code, "-miat-id") {
					if strings.HasSuffix(code, "-wiat-id") {
						mapClaims["iat"] = "1234abcd"
					} else {
						// 1970-01-01 00:16:40 +0000 UTC
						mapClaims["iat"] = 1000
					}
				}
				if !strings.HasSuffix(code, "-mexp-id") {
					if strings.HasSuffix(code, "-wexp-id") {
						mapClaims["exp"] = "1234abcd"
					} else {
						// 2096-10-02 07:06:40 +0000 UTC
						mapClaims["exp"] = 4000000000
					}
				}
				if !strings.HasSuffix(code, "-msub-id") {
					if strings.HasSuffix(code, "-wsub-id") {
						mapClaims["sub"] = "me"
					} else {
						mapClaims["sub"] = "myself"
					}
				}
				if strings.HasSuffix(code, "-wazp-id") {
					mapClaims["azp"] = "bar"
				} else if !strings.HasSuffix(code, "-mazp-id") {
					mapClaims["azp"] = "foo"
				}
				if strings.HasSuffix(code, "-wn-id") {
					nonce = nonce + "-wrong"
				}
				if !strings.HasSuffix(code, "-mn-id") {
					mapClaims["nonce"] = nonce
				}
				keyBytes, err := os.ReadFile("testdata/integration/files/pkcs8.key")
				if err != nil {
					errResp(err)
					return
				}

				key, parseErr := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
				if parseErr != nil {
					errResp(err)
					return
				}

				var kid string
				if strings.HasSuffix(code, "-wkid-id") {
					kid = "not-found"
				} else {
					kid = "rs256"
				}

				idToken, err := lib.CreateJWT("RS256", key, mapClaims, map[string]interface{}{"kid": kid})
				if err != nil {
					errResp(err)
					return
				}

				idTokenToAdd = `"id_token":"` + idToken + `",
				`
			}

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100,
				` + idTokenToAdd +
				`"form_params": "` + req.PostForm.Encode() + `",
				"authorization": "` + req.Header.Get("Authorization") + `"
			}`)
			_, werr := rw.Write(body)
			if werr != nil {
				t.Log(werr)
			}

			return
		} else if req.URL.Path == "/userinfo" {
			body := []byte(`{"sub": "myself"}`)
			_, werr := rw.Write(body)
			if werr != nil {
				t.Log(werr)
			}

			return
		} else if req.URL.Path == "/jwks" {
			jsonBytes, rerr := os.ReadFile("testdata/integration/files/jwks.json")
			if rerr != nil {
				errResp(rerr)
				return
			}
			b := bytes.NewBuffer(jsonBytes)
			_, werr := b.WriteTo(rw)
			if werr != nil {
				t.Log(werr)
			}

			return
		} else if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"token_endpoint": "http://` + req.Host + `/token",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			if werr != nil {
				t.Log(werr)
			}
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	type testCase struct {
		name          string
		filename      string
		method        string
		path          string
		header        http.Header
		status        int
		params        string
		authorization string
		wantErrLog    string
	}

	for _, tc := range []testCase{
		{"wrong method", "04_couper.hcl", http.MethodPost, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusForbidden, "", "", "access control error: ac: wrong method (POST)"},
		{"oidc: wrong method", "07_couper.hcl", http.MethodPost, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: wrong method (POST)"},
		{"no code, but error", "04_couper.hcl", http.MethodGet, "/cb?error=qeuboub", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"no code; error handler", "05_couper.hcl", http.MethodGet, "/cb?error=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusTeapot, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"oidc: no code; error handler", "10_couper.hcl", http.MethodGet, "/cb?error=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusTeapot, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"code, missing state param", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"st=qerbnr"}}, http.StatusForbidden, "", "", "access control error: ac: missing state query parameter; query=\"code=qeuboub\""},
		{"code, wrong state param", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=wrong", http.Header{"Cookie": []string{"st=" + st}}, http.StatusForbidden, "", "", "access control error: ac: state mismatch: \"wrong\" (from query param) vs. \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (verifier_value: \"qeirtbnpetrbi\")"},
		{"code, state param, wrong CSRF token", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st + "-wrong"}}, http.StatusForbidden, "", "", "access control error: ac: state mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (from query param) vs. \"Mj0ecDMNNzOwqUt1iFlY8TOTTKa17ISo8ARgt0pyb1A\" (verifier_value: \"qeirtbnpetrbi-wrong\")"},
		{"code, state param, missing CSRF token", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=" + state, http.Header{}, http.StatusForbidden, "", "", "access control error: ac: Empty verifier_value"},
		{"code, missing nonce", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-mn-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing nonce claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"sub\":\"myself\"}'"},
		{"code, wrong nonce", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wn-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: nonce mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ-wrong\" (from nonce claim) vs. \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (verifier_value: \"qeirtbnpetrbi\")"},
		{"code, nonce, wrong CSRF token", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st + "-wrong"}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: nonce mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (from nonce claim) vs. \"Mj0ecDMNNzOwqUt1iFlY8TOTTKa17ISo8ARgt0pyb1A\" (verifier_value: \"qeirtbnpetrbi-wrong\")"},
		{"code, nonce, missing CSRF token", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: Empty verifier_value"},
		{"code, missing sub claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-msub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing sub claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\"}'"},
		{"code, sub mismatch", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wsub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: subject mismatch, in ID token \"me\", in userinfo response \"myself\""},
		{"code, missing exp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-mexp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing exp claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, wrong exp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wexp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: json: unsupported type: string"},
		{"code, missing iat claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-miat-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing iat claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, wrong iat claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wiat-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: iat claim in ID token must be number, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":\"1234abcd\", \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, missing azp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-mazp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing azp claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, wrong azp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wazp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: azp claim / client ID mismatch, azp = \"bar\", client ID = \"foo\""},
		{"code, missing iss claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-miss-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token issuer is invalid: 'iss' value doesn't match expectation"},
		{"code, wrong iss claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wiss-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token issuer is invalid: 'iss' value doesn't match expectation"},
		{"code, missing aud claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-maud-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing aud claim in ID token, claims='jwt.MapClaims{\"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, null aud claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-naud-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: aud claim in ID token must not be null"},
		{"code, wrong aud claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-waud-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token audience is invalid: 'foo' wasn't found in aud claim"},
		{"code, wrong kid", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wkid-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token is unverifiable: Keyfunc returned an error"},
		{"code; client_secret_basic; PKCE", "04_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; client_secret_post", "05_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "client_id=foo&client_secret=etbinbp4in&code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "", ""},
		{"code, state param", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st}}, http.StatusOK, "code=qeuboub&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code, nonce param", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusOK, "code=qeuboub-id&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; client_secret_basic; PKCE; relative redirect_uri", "08_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}, "X-Forwarded-Proto": []string{"https"}, "X-Forwarded-Host": []string{"www.example.com"}}, http.StatusOK, "code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=https%3A%2F%2Fwww.example.com%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; nonce param; relative redirect_uri", "09_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}, "X-Forwarded-Proto": []string{"https"}, "X-Forwarded-Host": []string{"www.example.com"}}, http.StatusOK, "code=qeuboub-id&grant_type=authorization_code&redirect_uri=https%3A%2F%2Fwww.example.com%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			h := test.New(subT)

			shutdown, hook := newCouperWithTemplate("testdata/oauth2/"+tc.filename, h, map[string]interface{}{"asOrigin": oauthOrigin.URL})
			defer shutdown()

			req, err := http.NewRequest(tc.method, "http://back.end:8080"+tc.path, nil)
			h.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.status {
				subT.Errorf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
			}

			tokenResBytes, err := io.ReadAll(res.Body)
			h.Must(err)

			var jData map[string]interface{}
			_ = json.Unmarshal(tokenResBytes, &jData)

			if params, ok := jData["form_params"]; ok {
				if params != tc.params {
					subT.Errorf("%q: expected params %s, got: %s", tc.name, tc.params, params)
				}
			} else {
				if tc.params != "" {
					subT.Errorf("%q: expected params %s, got no", tc.name, tc.params)
				}
			}
			if authorization, ok := jData["authorization"]; ok {
				if tc.authorization != authorization {
					subT.Errorf("%q: expected authorization %s, got: %s", tc.name, tc.authorization, authorization)
				}
			} else {
				if tc.authorization != "" {
					subT.Errorf("%q: expected authorization %s, got no", tc.name, tc.authorization)
				}
			}

			message := getFirstAccessLogMessage(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					subT.Errorf("%q: Expected error log: %q, actual: %#v", tc.name, tc.wantErrLog, message)
				}
			} else {
				if !strings.HasPrefix(message, tc.wantErrLog) {
					subT.Errorf("%q: Expected error log message: %q, actual: %#v", tc.name, tc.wantErrLog, message)
				}
			}
		})
	}
}

func TestOAuth2_AC_Backend(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	// authorization server creates token response with sub property, JWT ID token with sub claim and userinfo response with sub property from X-Sub request header
	asOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		sub := req.Header.Get("X-Sub")
		if req.URL.Path == "/token" {
			if req.Method != http.MethodPost {
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			mapClaims := jwt.MapClaims{
				"iss": "https://authorization.server",
				"aud": "foo",
				"sub": "myself",
				"exp": 4000000000,
				"iat": 1000,
			}
			keyBytes, err := os.ReadFile("testdata/integration/files/pkcs8.key")
			helper.Must(err)
			key, parseErr := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
			helper.Must(parseErr)
			idToken, err := lib.CreateJWT("RS256", key, mapClaims, map[string]interface{}{"kid": "rs256"})
			helper.Must(err)
			// idToken, _ := lib.CreateJWT("HS256", []byte("$e(rEt"), mapClaims, nil)
			idTokenToAdd := `"id_token":"` + idToken + `",
			`

			body := []byte(`{
				"access_token": "abcdef0123456789",
				` + idTokenToAdd +
				`"sub": "` + sub + `"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/userinfo" {
			rw.Header().Set("Content-Type", "application/json")
			body := []byte(`{"sub": "` + sub + `"}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/jwks" {
			rw.Header().Set("Content-Type", "application/json")
			jsonBytes, rerr := os.ReadFile("testdata/integration/files/jwks.json")
			helper.Must(rerr)
			b := bytes.NewBuffer(jsonBytes)
			_, werr := b.WriteTo(rw)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/.well-known/openid-configuration" {
			rw.Header().Set("Content-Type", "application/json")
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer asOrigin.Close()

	shutdown, hook := newCouperWithTemplate("testdata/oauth2/11_couper.hcl", helper, map[string]interface{}{"asOrigin": asOrigin.URL})
	defer shutdown()

	type backendExpectation struct {
		path, name string
	}

	type testCase struct {
		name string
		path string
		exp  backendExpectation
	}

	time.Sleep(time.Second * 2) // wait for all oidc/jwks inits
	//for _, entry := range hook.AllEntries() {
	//	println(entry.String())
	//}
	//hook.Reset()

	for _, tc := range []testCase{
		{"OAuth2 Authorization Code, referenced backend", "/oauth1/redir?code=qeuboub", backendExpectation{"/token", "token"}},
		{"OAuth2 Authorization Code, inline backend", "/oauth2/redir?code=qeuboub", backendExpectation{"/token", "anonymous_56_5_token_endpoint"}},
		{"OIDC Authorization Code, referenced backend", "/oidc1/redir?code=qeuboub", backendExpectation{"/token", "token"}},
		{"OIDC Authorization Code, referenced backends", "/oidc1.1/redir?code=qeuboub", backendExpectation{"/token", "token"}},
		{"OIDC Authorization Code, inline backend", "/oidc2/redir?code=qeuboub", backendExpectation{"/token", "anonymous_98_20_token_backend"}},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			h.Must(err)

			req.Header.Set("Cookie", "pkcecv=qerbnr")

			hook.Reset()
			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusOK {
				subT.Fatalf("expected Status %d, got: %d", http.StatusOK, res.StatusCode)
			}
			defer res.Body.Close()

			tokenResBytes, err := io.ReadAll(res.Body)
			h.Must(err)

			var jData map[string]interface{}
			h.Must(json.Unmarshal(tokenResBytes, &jData))
			if sub, ok := jData["sub"]; ok {
				if sub != "myself" {
					subT.Errorf("expected sub %q, got: %q", "myself", sub)
				}
			} else {
				subT.Errorf("expected sub %q, got no", "myself")
			}

			var seen bool
			for _, entry := range hook.AllEntries() {
				if entry.Data["type"] == "couper_backend" && entry.Data["backend"] != "" {
					if backend, ok := entry.Data["backend"].(string); ok {
						if request, ok := entry.Data["request"]; ok {
							path, _ := request.(logging.Fields)["path"].(string)
							if reflect.DeepEqual(tc.exp, backendExpectation{
								path, backend,
							}) {
								seen = true
								break
							}
						}

					}
				}
			}

			if !seen {
				subT.Errorf("expected %#v, got %q", tc.exp, getUpstreamLogBackendName(hook))
			}
		})
	}
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", fmt.Errorf("bearer required with authorization header")
}

func TestOAuth2_CC_Backend(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	// authorization server creates JWT access token with sub-claim from X-Sub request header
	asOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		sub := req.Header.Get("X-Sub")
		if req.URL.Path == "/token" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			mapClaims := jwt.MapClaims{"sub": sub}
			accessToken, _ := lib.CreateJWT("HS256", []byte("$e(rEt"), mapClaims, nil)
			body := []byte(`{"access_token": "` + accessToken + `"}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer asOrigin.Close()

	// resource server sends value of sub claim of JWT bearer token
	rsOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		authorization := req.Header.Get("Authorization")
		tokenString, err := getBearer(authorization)
		helper.Must(err)
		jwtParser := jwt.NewParser()
		claims := jwt.MapClaims{}
		_, _, err = jwtParser.ParseUnverified(tokenString, claims)
		helper.Must(err)
		sub := claims["sub"].(string)

		rw.Header().Set("X-Sub2", sub)
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer rsOrigin.Close()

	shutdown, hook := newCouperWithTemplate("testdata/oauth2/14_couper.hcl", helper, map[string]interface{}{"asOrigin": asOrigin.URL, "rsOrigin": rsOrigin.URL})
	defer shutdown()

	type testCase struct {
		name        string
		path        string
		backendName string
	}

	for _, tc := range []testCase{
		{"referenced backend", "/rs1", "token"},
		{"inline backend", "/rs2", "anonymous_32_12"},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			h.Must(err)

			hook.Reset()
			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusNoContent {
				subT.Errorf("expected Status %d, got: %d", http.StatusNoContent, res.StatusCode)
			}

			sub := res.Header.Get("X-Sub2")
			if sub != "myself" {
				subT.Errorf("expected sub %q, got: %q", "myself", sub)
			}

			backendName := getUpstreamLogBackendName(hook)
			if backendName != tc.backendName {
				subT.Errorf("expected backend name %q, got: %q", tc.backendName, backendName)
			}
		})
	}
}

func getUpstreamLogBackendName(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_backend" && entry.Data["backend"] != "" {
			if backend, ok := entry.Data["backend"].(string); ok {
				return backend
			}
		}
	}

	return ""
}

func TestOAuth2_Locking(t *testing.T) {
	helper := test.New(t)
	client := test.NewHTTPClient()

	token := "token-"
	var oauthRequestCount uint32

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/oauth2" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		atomic.AddUint32(&oauthRequestCount, 1)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		n := fmt.Sprintf("%d", atomic.LoadUint32(&oauthRequestCount))
		body := []byte(`{
				"access_token": "` + token + n + `",
				"token_type": "bearer",
				"expires_in": 1.5
			}`)

		// Slow down token request
		time.Sleep(time.Second)

		_, werr := rw.Write(body)
		if werr != nil {
			t.Error(werr)
		}
	}))
	defer oauthOrigin.Close()

	ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/resource" {
			if auth := req.Header.Get("Authorization"); auth != "" {
				rw.Header().Set("Token", auth[len("Bearer "):])
				rw.WriteHeader(http.StatusNoContent)
			}

			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
	defer ResourceOrigin.Close()

	confPath := "testdata/oauth2/1_retries_couper.hcl"
	shutdown, hook := newCouperWithTemplate(
		confPath, helper, map[string]interface{}{
			"asOrigin": oauthOrigin.URL,
			"rsOrigin": ResourceOrigin.URL,
		},
	)
	defer shutdown()

	req, rerr := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(rerr)

	hook.Reset()

	req.URL.Path = "/"

	var responses []*http.Response
	var wg sync.WaitGroup

	addLock := &sync.Mutex{}
	// Fire 5 requests in parallel...
	waitCh := make(chan struct{})
	errors := make(chan error, 5)
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			<-waitCh
			res, e := client.Do(req)
			if e != nil {
				errors <- e
				return
			}

			addLock.Lock()
			responses = append(responses, res)
			addLock.Unlock()

		}()
	}
	close(waitCh)
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}

	for _, res := range responses {
		if res.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status NoContent, got: %d", res.StatusCode)
		}

		if token+"1" != res.Header.Get("Token") {
			t.Errorf("Invalid token given: want %s1, got: %s", token, res.Header.Get("Token"))
		}
	}

	if count := atomic.LoadUint32(&oauthRequestCount); count != 1 {
		t.Errorf("OAuth2 requests: want 1, got: %d", count)
	}

	t.Run("Lock is effective", func(subT *testing.T) {
		// Wait until token has expired.
		time.Sleep(2 * time.Second)

		// Fetch new token.
		go func() {
			res, err := client.Do(req)
			if err != nil {
				subT.Error(err)
				return
			}

			if token+"2" != res.Header.Get("Token") {
				subT.Errorf("Received wrong token: want %s2, got: %s", token, res.Header.Get("Token"))
			}
		}()

		// Slow response due to lock
		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			subT.Error(err)
			return
		}

		timeElapsed := time.Since(start)

		if token+"2" != res.Header.Get("Token") {
			subT.Errorf("Received wrong token: want %s2, got: %s", token, res.Header.Get("Token"))
		}

		if timeElapsed < time.Second {
			subT.Errorf("Response came too fast: dysfunctional lock?! (%s)", timeElapsed.String())
		}
	})

	t.Run("Mem store expiry", func(subT *testing.T) {
		// Wait again until token has expired.
		time.Sleep(2 * time.Second)
		h := test.New(subT)
		// Request fresh token and store in memstore
		res, err := client.Do(req)
		h.Must(err)

		if res.StatusCode != http.StatusNoContent {
			subT.Errorf("Unexpected response status: want %d, got: %d", http.StatusNoContent, res.StatusCode)
		}

		if token+"3" != res.Header.Get("Token") {
			subT.Errorf("Received wrong token: want %s3, got: %s", token, res.Header.Get("Token"))
		}

		if count := atomic.LoadUint32(&oauthRequestCount); count != 3 {
			subT.Errorf("Unexpected number of OAuth2 requests: want 3, got: %d", count)
		}

		// Disconnect OAuth server
		oauthOrigin.Close()

		// Next request gets token from memstore
		res, err = client.Do(req)
		h.Must(err)
		if res.StatusCode != http.StatusNoContent {
			subT.Errorf("Unexpected response status: want %d, got: %d", http.StatusNoContent, res.StatusCode)
		}

		if token+"3" != res.Header.Get("Token") {
			subT.Errorf("Wrong token from mem store: want %s3, got: %s", token, res.Header.Get("Token"))
		}

		// Wait until token has expired. Next request accesses the OAuth server again.
		time.Sleep(2 * time.Second)
		res, err = newClient().Do(req)
		h.Must(err)
		if res.StatusCode != http.StatusBadGateway {
			subT.Errorf("Unexpected response status: want %d, got: %d", http.StatusBadGateway, res.StatusCode)
		}
	})
}

func TestNestedBackendOauth2(t *testing.T) {
	helper := test.New(t)
	shutdown, hook := newCouperMultiFiles("testdata/oauth2/15_couper.hcl", "", helper)
	defer shutdown()

	time.Sleep(time.Second / 2)

	logs := hook.AllEntries()
	for _, log := range logs {
		if log.Level == logrus.ErrorLevel {
			t.Error(log.String())
		}
	}
}

func TestTokenRequest(t *testing.T) {
	helper := test.New(t)

	asOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("KeyId") != "the-key" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		if user, _, _ := req.BasicAuth(); user != "the-key" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		reqBody, _ := io.ReadAll(req.Body)

		// path_prefix context test prepends "the-key"

		if req.URL.Path == "/the-key/token" {
			expBody := "grant_type=client_credentials"
			if expBody != string(reqBody) {
				t.Errorf("wrong request body /token\nwant: %q\ngot:  %q", expBody, reqBody)
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "tok0",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/the-key/token1" {
			expBody := "client_id=clid&client_secret=cls&grant_type=client_credentials"
			if expBody != string(reqBody) {
				t.Errorf("wrong request body /token1\nwant: %q\ngot:  %q", expBody, reqBody)
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "tok1",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/the-key/token2" {
			if req.URL.RawQuery != "foo=bar" {
				t.Errorf("wrong request URL query /token2\nwant: %q\ngot:  %q", "foo=bar", req.URL.RawQuery)
			}
			expBody := "client_id=clid&client_secret=cls&grant_type=password&password=asdf&username=user"
			if expBody != string(reqBody) {
				t.Errorf("wrong request body /token2\nwant: %q\ngot:  %q", expBody, reqBody)
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "tok2",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer asOrigin.Close()

	rsOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer tok0" ||
			req.Header.Get("Auth-1") != "tok1" ||
			req.Header.Get("Auth-2") != "tok2" ||
			req.Header.Get("Auth-3") != "tok2" ||
			req.Header.Get("Auth-4") != "tok1" ||
			req.Header.Get("Auth-5") != "tok2" ||
			req.Header.Get("Auth-6") != "tok2" ||
			req.Header.Get("KeyId") != "the-key" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		if req.URL.Path == "/resource" {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
	defer rsOrigin.Close()

	vaultOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/key" {
			rw.WriteHeader(http.StatusOK)

			body := []byte("the-key")
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer vaultOrigin.Close()

	confPath := "testdata/oauth2/token_request.hcl"
	shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"asOrigin": asOrigin.URL, "rsOrigin": rsOrigin.URL, "vaultOrigin": vaultOrigin.URL})
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/resource", nil)
	helper.Must(err)
	hook.Reset()
	res, err := newClient().Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got: %d", http.StatusNoContent, res.StatusCode)
	}
}

func TestTokenRequest_Config_Errors(t *testing.T) {
	type testCase struct {
		name  string
		hcl   string
		error string
	}

	for _, tc := range []testCase{
		{
			"invalid label",
			`server {}
definitions {
  backend "be" {
    beta_token_request "the label" {
      url = "http://localhost:8082/token2"
      token = beta_token_response.json_body.tok
      ttl = "1m"
    }
  }
}
`,
			"couper.hcl:4,24-35: label contains invalid character(s), allowed are 'a-z', 'A-Z', '0-9' and '_';",
		},
		{
			"multiple default labels (LabelRanges)",
			`server {}
definitions {
  backend "be" {
    beta_token_request {
      url = "http://localhost:8081/token1"
      token = beta_token_response.json_body.tok
      ttl = "1m"
    }
    beta_token_request "default" {
      url = "http://localhost:8082/token2"
      token = beta_token_response.json_body.tok
      ttl = "2m"
    }
  }
}
`,
			"couper.hcl:9,24-33: token request names (either default or explicitly set via label) must be unique: \"default\";",
		},
		{
			"multiple default labels (DefRange)",
			`server {}
definitions {
  backend "be" {
    beta_token_request "default" {
      url = "http://localhost:8081/token1"
      token = beta_token_response.json_body.tok
      ttl = "1m"
    }
    beta_token_request {
      url = "http://localhost:8082/token2"
      token = beta_token_response.json_body.tok
      ttl = "2m"
    }
  }
}
`,
			"couper.hcl:9,5-23: token request names (either default or explicitly set via label) must be unique: \"default\";",
		},
		{
			"multiple default labels (inline backend)",
			`
server {
  endpoint "/" {
    proxy {
      backend {
        beta_token_request "default" {
          url = "http://localhost:8081/token1"
          token = beta_token_response.json_body.tok
          ttl = "1m"
        }
        beta_token_request {
          url = "http://localhost:8082/token2"
          token = beta_token_response.json_body.tok
          ttl = "2m"
        }
      }
    }
  }
}
`,
			"couper.hcl:11,9-27: token request names (either default or explicitly set via label) must be unique: \"default\";",
		},
		{
			"multiple labels",
			`server {}
definitions {
  backend "be" {
    beta_token_request "a" {
      url = "http://localhost:8081/token1"
      token = beta_token_response.json_body.tok
      ttl = "1m"
    }
    beta_token_request "a" {
      url = "http://localhost:8082/token2"
      token = beta_token_response.json_body.tok
      ttl = "2m"
    }
  }
}
`,
			"couper.hcl:9,24-27: token request names (either default or explicitly set via label) must be unique: \"a\"; ",
		},
		{
			"multiple labels (inline backend)",
			`
server {
   endpoint "/" {
     proxy {
       backend {
         beta_token_request "a" {
          url = "http://localhost:8081/token1"
          token = beta_token_response.json_body.tok
          ttl = "1m"
        }
        beta_token_request "a" {
          url = "http://localhost:8082/token2"
          token = beta_token_response.json_body.tok
          ttl = "2m"
        }
      }
    }
  }
}
`,
			"couper.hcl:11,28-31: token request names (either default or explicitly set via label) must be unique: \"a\"; ",
		},
	} {
		var errMsg string
		_, err := configload.LoadBytes([]byte(tc.hcl), "couper.hcl")
		if err != nil {
			if _, ok := err.(errors.GoError); ok {
				errMsg = err.(errors.GoError).LogError()
			} else {
				errMsg = err.Error()
			}
		}

		if !strings.HasPrefix(errMsg, tc.error) {
			t.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tc.name, tc.error, errMsg)
		}
	}
}

func TestTokenRequest_Runtime_Errors(t *testing.T) {
	helper := test.New(t)

	asOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer asOrigin.Close()

	type testCase struct {
		name       string
		filename   string
		wantStatus int
		wantErrLog string
	}

	for _, tc := range []testCase{
		{"token request error, handled by error handler", "01_token_request_error.hcl", http.StatusNoContent, "backend error: be: request error: tr: token request failed"},
		{"token expression evaluation error", "02_token_request_error.hcl", http.StatusBadGateway, "couper-bytes.hcl:23,15-31: Call to unknown function; There is no function named \"evaluation_error\"."},
		{"null token", "03_token_request_error.hcl", http.StatusBadGateway, "backend error: be: request error: tr: token expression evaluates to null"},
		{"non-string token", "04_token_request_error.hcl", http.StatusBadGateway, "backend error: be: request error: tr: token expression must evaluate to a string"},
		{"ttl expression evaluation error", "05_token_request_error.hcl", http.StatusBadGateway, "couper-bytes.hcl:24,13-29: Call to unknown function; There is no function named \"evaluation_error\"."},
		{"null ttl", "06_token_request_error.hcl", http.StatusBadGateway, "backend error: be: request error: tr: ttl expression evaluates to null"},
		{"non-string ttl", "07_token_request_error.hcl", http.StatusBadGateway, "backend error: be: request error: tr: ttl expression must evaluate to a string"},
		{"non-duration ttl", "08_token_request_error.hcl", http.StatusBadGateway, "backend error: be: request error: tr: ttl: time: invalid duration \"no duration\""},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			shutdown, hook := newCouperWithTemplate("testdata/oauth2/"+tc.filename, h, map[string]interface{}{"asOrigin": asOrigin.URL})
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/resource", nil)
			h.Must(err)
			hook.Reset()
			res, err := newClient().Do(req)
			h.Must(err)

			if res.StatusCode != tc.wantStatus {
				subT.Errorf("expected status %d, got: %d", tc.wantStatus, res.StatusCode)
			}

			message := getFirstAccessLogMessage(hook)
			if message != tc.wantErrLog {
				subT.Errorf("error log\nwant: %q\ngot:  %q", tc.wantErrLog, message)
			}

			shutdown()
		})
	}

	asOrigin.Close()
}
