package accesscontrol_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/avenga/couper/accesscontrol"
	acjwt "github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/test"
)

func Test_JWT_NewJWT_RSA(t *testing.T) {
	helper := test.New(t)

	type fields struct {
		algorithm      string
		claims         hcl.Expression
		claimsRequired []string
		pubKey         []byte
		pubKeyPath     string
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	helper.Must(err)

	bytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	helper.Must(err)

	pubKeyBytesPKIX := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: bytes,
	})
	pubKeyBytesPKCS1 := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	})
	privKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})
	// created using
	// openssl req -new -newkey rsa:1024 -days 100000 -nodes -x509
	certBytes := []byte(`-----BEGIN CERTIFICATE-----
MIICaDCCAdGgAwIBAgIUZe+V/eBcYEaoORX8mfsyR8LqY/kwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAgFw0yMTA0MTIxMzI1MzRaGA8yMjk1
MDEyNjEzMjUzNFowRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUx
ITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDCBnzANBgkqhkiG9w0B
AQEFAAOBjQAwgYkCgYEA2m79uRP+f/L6YgCuQoAiY6Qs5pccKR4DNfb+vQOsO+xx
ZxWrY3RLSLOYKCBybHClz0JLT61duq7yfOl+03lYE6wTdy5XN1PGoijITj3cA6g1
Eah6/CirrDVqEVIng+5lsw/Qws1gOOkHaCdfkL85Trm4AWqppgFgIc/wafHZjekC
AwEAAaNTMFEwHQYDVR0OBBYEFCAUN20ma8sVaz1KZttyofv6tDZdMB8GA1UdIwQY
MBaAFCAUN20ma8sVaz1KZttyofv6tDZdMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZI
hvcNAQELBQADgYEADyu05JNvWly50lvUksx85QwEMb7oZha6aov/9eslJnHD10Zu
QolLGgj3tz4NbDEitq+zKMr0uTHvP1Vyu1mXAflcpYcJA4ZmuB3Oj39e0U0gnmr/
1T2dX1uHaAWl3pCmkRH1Dmpsx2sHllN/yizHpve2rrVpM9ZMXEdPxnzNNFE=
-----END CERTIFICATE-----`)

	for _, signingMethod := range []jwt.SigningMethod{
		jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512,
	} {
		alg := signingMethod.Alg()
		tests := []struct {
			name    string
			fields  fields
			wantErr string
		}{
			{"missing key-file path", fields{}, "configuration error: jwt key: read error: required: configured attribute or file"},
			{"missing key-file", fields{pubKeyPath: "./not-there.file"}, "not-there.file: no such file or directory"},
			{"PKIX", fields{
				algorithm: alg,
				pubKey:    pubKeyBytesPKIX,
			}, ""},
			{"PKCS1", fields{
				algorithm: alg,
				pubKey:    pubKeyBytesPKCS1,
			}, ""},
			{"Cert", fields{
				algorithm: alg,
				pubKey:    certBytes,
			}, ""},
			{"Priv", fields{
				algorithm: alg,
				pubKey:    privKeyBytes,
			}, "key is not a valid RSA public key"},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v / %s", signingMethod, tt.name), func(subT *testing.T) {
				key, rerr := reader.ReadFromAttrFile("jwt key", string(tt.fields.pubKey), tt.fields.pubKeyPath)
				if rerr != nil {
					logErr := rerr.(errors.GoError)
					if tt.wantErr != "" && !strings.HasSuffix(logErr.LogError(), tt.wantErr) {
						subT.Errorf("\nWant:\t%q\nGot:\t%q", tt.wantErr, logErr.LogError())
					} else if tt.wantErr == "" {
						subT.Fatal(logErr.LogError())
					}
					return
				}

				j, jerr := ac.NewJWT(&ac.JWTOptions{
					Algorithm:      tt.fields.algorithm,
					Claims:         tt.fields.claims,
					ClaimsRequired: tt.fields.claimsRequired,
					Name:           "test_ac",
					Key:            key,
					Source:         ac.NewJWTSource("", "Authorization"),
				})
				if jerr != nil {
					if tt.wantErr != jerr.Error() {
						subT.Errorf("error: %v, want: %v", jerr.Error(), tt.wantErr)
					}
				} else if tt.wantErr != "" {
					subT.Errorf("error expected: %v", tt.wantErr)
				}
				if tt.wantErr == "" && j == nil {
					subT.Errorf("JWT struct expected")
				}
			})
		}
	}
}

func Test_JWT_Validate(t *testing.T) {
	type fields struct {
		algorithm      acjwt.Algorithm
		claims         map[string]string
		claimsRequired []string
		source         ac.JWTSource
		pubKey         []byte
	}

	for _, signingMethod := range []jwt.SigningMethod{
		jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512,
		jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512,
	} {

		pubKeyBytes, privKey := newRSAKeyPair()

		tok := jwt.NewWithClaims(signingMethod, jwt.MapClaims{
			"aud":     "peter",
			"test123": "value123",
		})
		var token string
		var tokenErr error

		algo := acjwt.NewAlgorithm(signingMethod.Alg())

		if algo.IsHMAC() {
			pubKeyBytes = []byte("mySecretK3y")
			token, tokenErr = tok.SignedString(pubKeyBytes)
		} else {
			token, tokenErr = tok.SignedString(privKey)
		}

		if tokenErr != nil {
			t.Error(tokenErr)
		}

		tests := []struct {
			name    string
			fields  fields
			req     *http.Request
			wantErr bool
		}{
			{"src: header /w empty bearer", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization"),
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), true},
			{"src: header /w valid bearer", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization"),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), false},
			{"src: header /w no cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", ""),
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), true},
			{"src: header /w empty cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", ""),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", ""), true},
			{"src: header /w valid cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", ""),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", token), false},
			{"src: header /w valid bearer & claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":     "peter",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				source:         ac.NewJWTSource("", "Authorization"),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), false},
			{"src: header /w valid bearer & w/o claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":  "peter",
					"cptn": "hook",
				},
				source: ac.NewJWTSource("", "Authorization"),
				pubKey: pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), true},
			{"src: header /w valid bearer & w/o required claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud": "peter",
				},
				claimsRequired: []string{"exp"},
				source:         ac.NewJWTSource("", "Authorization"),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), true},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v_%s", signingMethod, tt.name), func(t *testing.T) {
				claimValMap := make(map[string]cty.Value)
				for k, v := range tt.fields.claims {
					claimValMap[k] = cty.StringVal(v)
				}
				j, err := ac.NewJWT(&ac.JWTOptions{
					Algorithm:      tt.fields.algorithm.String(),
					Claims:         hcl.StaticExpr(cty.ObjectVal(claimValMap), hcl.Range{}),
					ClaimsRequired: tt.fields.claimsRequired,
					Name:           "test_ac",
					Source:         tt.fields.source,
					Key:            tt.fields.pubKey,
				})
				if err != nil {
					t.Error(err)
					return
				}

				if err = j.Validate(tt.req); (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}

				if !tt.wantErr && tt.fields.claims != nil {
					acMap := tt.req.Context().Value(request.AccessControls).(map[string]interface{})
					if claims, ok := acMap["test_ac"]; !ok {
						t.Errorf("Expected a configured access control name within request context")
					} else {
						claimsMap := claims.(map[string]interface{})
						for k, v := range tt.fields.claims {
							if claimsMap[k] != v {
								t.Errorf("Claim does not match: %q want: %v, got: %v", k, v, claimsMap[k])
							}
						}
					}

				}
			})
		}
	}
}

func Test_JWT_yields_scopes(t *testing.T) {
	signingMethod := jwt.SigningMethodHS256
	algo := acjwt.NewAlgorithm(signingMethod.Alg())
	expScopes := []string{"foo", "bar"}

	tests := []struct {
		name       string
		scopeClaim string
		scope      interface{}
		wantErr    bool
	}{
		{
			"space-separated list",
			"scp",
			"foo bar",
			false,
		},
		{
			"list of string",
			"scoop",
			[]string{"foo", "bar"},
			false,
		},
		{
			"error: boolean",
			"scope",
			true,
			true,
		},
		{
			"error: number",
			"scope",
			1.23,
			true,
		},
		{
			"error: list of bool",
			"scope",
			[]bool{true, false},
			true,
		},
		{
			"error: list of number",
			"scope",
			[]int{1, 2},
			true,
		},
		{
			"error: mixed list",
			"scope",
			[]interface{}{"eins", 2},
			true,
		},
		{
			"error: object",
			"scope",
			map[string]interface{}{"foo": 1, "bar": 1},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{}
			claims[tt.scopeClaim] = tt.scope
			tok := jwt.NewWithClaims(signingMethod, claims)
			pubKeyBytes := []byte("mySecretK3y")
			token, tokenErr := tok.SignedString(pubKeyBytes)
			if tokenErr != nil {
				t.Error(tokenErr)
			}

			source := ac.NewJWTSource("", "Authorization")
			j, err := ac.NewJWT(&ac.JWTOptions{
				Algorithm:  algo.String(),
				Name:       "test_ac",
				ScopeClaim: tt.scopeClaim,
				Source:     source,
				Key:        pubKeyBytes,
			})
			if err != nil {
				t.Error(err)
				return
			}

			req := setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)

			if err = j.Validate(req); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				scopesList, ok := req.Context().Value(request.Scopes).([]string)
				if !ok {
					t.Errorf("Expected scopes within request context")
				} else {
					for i, v := range expScopes {
						if scopesList[i] != v {
							t.Errorf("Scopes do not match, want: %v, got: %v", v, scopesList[i])
						}
					}
				}

			}
		})
	}
}

func TestJwtConfig(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"missing definition for access_control",
			`
			server "test" {
			  access_control = ["myac"]
			}
			`,
			"", // FIXME Missing myac
		},
		{
			"missing both signature_algorithm/jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			  }
			}
			`,
			"configuration error", // signature_algorithm or jwks_url required
		},
		{
			"signature_algorithm, missing key/key_file",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
				header = "..."
			  }
			}
			`,
			"configuration error", // required: configured attribute or file
		},
		{
			"signature_algorithm + key",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
				header = "..."
				key = "..."
			  }
			}
			`,
			"",
		},
		{
			"signature_algorithm + key_file",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
				header = "..."
				key_file = "testdata/secret.txt"
			  }
			}
			`,
			"",
		},
		{
			"jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				jwks_uri = "http://..."
				header = "..."
			  }
			}
			`,
			"",
		},
		{
			"signature_algorithm + jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				signature_algorithm = "HS256"
				jwks_uri = "http://..."
				header = "..."
			  }
			}
			`,
			"configuration error", // signature_algorithm cannot be used together with jwks_url
		},
		{
			"key + jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				key = "..."
				jwks_uri = "http://..."
				header = "..."
			  }
			}
			`,
			"configuration error", // key cannot be used together with jwks_url
		},
		{
			"key_file + jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				key_file = "..."
				jwks_uri = "http://..."
				header = "..."
			  }
			}
			`,
			"configuration error", // key_file cannot be used together with jwks_url
		},
		{
			"backend reference, missing jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				backend = "..."
				header = "..."
			  }
			}
			`,
			"configuration error", // backend requires jwks_uri
		},
		{
			"inline backend block, missing jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				backend {
				}
				header = "..."
			  }
			}
			`,
			"configuration error", // backend requires jwks_uri
		},
		{
			"inline backend block, missing jwks_uri",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
				backend = "foo"
				backend {
				}
				header = "..."
			  }
			}
			`,
			"configuration error", // backend must be either block or attribute
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			conf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if conf != nil {
				_, err = runtime.NewServerConfiguration(conf, nil, nil)
			}

			var error = ""
			if err != nil {
				error = err.Error()
			}

			if (tt.error == "" && error != "") || !strings.Contains(error, tt.error) {
				t.Errorf("Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.error, error)
			}
		})
	}
}

func newRSAKeyPair() (pubKeyBytes []byte, privKey *rsa.PrivateKey) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	if e := privKey.Validate(); e != nil {
		panic(e)
	}

	pubKeyBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	})
	return
}

func setCookieAndHeader(req *http.Request, key, value string) *http.Request {
	req.Header.Set(key, value)
	req.Header.Set("Cookie", key+"="+value)
	return req
}

func setContext(req *http.Request) *http.Request {
	evalCtx := eval.ContextFromRequest(req)
	*req = *req.WithContext(evalCtx.WithClientRequest(req))
	return req
}
