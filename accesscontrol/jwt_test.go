package accesscontrol_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
					Source:         ac.NewJWTSource("", "Authorization", nil),
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
	log, _ := test.NewLogger()
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
				source:    ac.NewJWTSource("", "Authorization", nil),
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), true},
			{"src: header /w valid bearer", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), false},
			{"src: header /w no cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", "", nil),
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), true},
			{"src: header /w empty cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", "", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", ""), true},
			{"src: header /w valid cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", "", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", token), false},
			{"src: header /w valid bearer & claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":     "peter",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				source:         ac.NewJWTSource("", "Authorization", nil),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), false},
			{"src: header /w valid bearer & w/o claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":  "peter",
					"cptn": "hook",
				},
				source: ac.NewJWTSource("", "Authorization", nil),
				pubKey: pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), true},
			{"src: header /w valid bearer & w/o required claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud": "peter",
				},
				claimsRequired: []string{"exp"},
				source:         ac.NewJWTSource("", "Authorization", nil),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), true},
			{
				"token_value number",
				fields{
					algorithm: algo,
					source:    ac.NewJWTSource("", "", hcltest.MockExprLiteral(cty.NumberIntVal(42))),
					pubKey:    pubKeyBytes,
				},
				setContext(httptest.NewRequest(http.MethodGet, "/", nil)),
				true,
			},
			{
				"token_value string",
				fields{
					algorithm:      algo,
					claims:         map[string]string{"aud": "peter", "test123": "value123"},
					claimsRequired: []string{"aud", "test123"},
					source:         ac.NewJWTSource("", "", hcltest.MockExprLiteral(cty.StringVal(token))),
					pubKey:         pubKeyBytes,
				},
				setContext(httptest.NewRequest(http.MethodGet, "/", nil)),
				false,
			},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v_%s", signingMethod, tt.name), func(subT *testing.T) {
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
					subT.Error(err)
					return
				}

				tt.req = tt.req.WithContext(context.WithValue(context.Background(), request.LogEntry, log.WithContext(context.Background())))

				if err = j.Validate(tt.req); (err != nil) != tt.wantErr {
					subT.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}

				if !tt.wantErr && tt.fields.claims != nil {
					acMap := tt.req.Context().Value(request.AccessControls).(map[string]interface{})
					if claims, ok := acMap["test_ac"]; !ok {
						subT.Errorf("Expected a configured access control name within request context")
					} else {
						claimsMap := claims.(map[string]interface{})
						for k, v := range tt.fields.claims {
							if claimsMap[k] != v {
								subT.Errorf("Claim does not match: %q want: %v, got: %v", k, v, claimsMap[k])
							}
						}
					}

				}
			})
		}
	}
}

func Test_JWT_yields_scopes(t *testing.T) {
	log, hook := test.NewLogger()
	signingMethod := jwt.SigningMethodHS256
	algo := acjwt.NewAlgorithm(signingMethod.Alg())

	rolesMap := map[string][]string{
		"admin": {"foo", "bar", "baz"},
		"user1": {"foo"},
		"user2": {"bar"},
		"*":     {"default"},
	}
	var noScope []string

	tests := []struct {
		name       string
		scopeClaim string
		scope      interface{}
		rolesClaim string
		roles      interface{}
		expWarning string
		expScopes  []string
	}{
		{
			"no scope, no roles",
			"scp",
			nil,
			"roles",
			nil,
			"",
			noScope,
		},
		{
			"scope: space-separated list",
			"scp",
			"foo bar",
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"scope: space-separated list, multiple",
			"scp",
			"foo bar foo",
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"scope: list of string",
			"scoop",
			[]string{"foo", "bar"},
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"scope: list of string, multiple",
			"scoop",
			[]string{"foo", "bar", "bar"},
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"scope: warn: boolean",
			"scope",
			true,
			"",
			nil,
			"invalid scope claim value type, ignoring claim, value: true",
			noScope,
		},
		{
			"scope: warn: number",
			"scope",
			1.23,
			"",
			nil,
			"invalid scope claim value type, ignoring claim, value: 1.23",
			noScope,
		},
		{
			"scope: warn: list of bool",
			"scope",
			[]bool{true, false},
			"",
			nil,
			"invalid scope claim value type, ignoring claim, value: []interface {}{true, false}",
			noScope,
		},
		{
			"scope: warn: list of number",
			"scope",
			[]int{1, 2},
			"",
			nil,
			"invalid scope claim value type, ignoring claim, value: []interface {}{1, 2}",
			noScope,
		},
		{
			"scope: warn: mixed list",
			"scope",
			[]interface{}{"eins", 2},
			"",
			nil,
			`invalid scope claim value type, ignoring claim, value: []interface {}{"eins", 2}`,
			noScope,
		},
		{
			"scope: warn: object",
			"scope",
			map[string]interface{}{"foo": 1, "bar": 1},
			"",
			nil,
			`invalid scope claim value type, ignoring claim, value: map[string]interface {}{"bar":1, "foo":1}`,
			noScope,
		},
		{
			"roles: single string",
			"",
			nil,
			"roles",
			"admin",
			"",
			[]string{"foo", "bar", "baz", "default"},
		},
		{
			"roles: space-separated list",
			"",
			nil,
			"roles",
			"user1 user2",
			"",
			[]string{"foo", "bar", "default"},
		},
		{
			"roles: space-separated list, multiple",
			"",
			nil,
			"roles",
			"user1 user2 user1",
			"",
			[]string{"foo", "bar", "default"},
		},
		{
			"roles: list of string",
			"",
			nil,
			"rollen",
			[]string{"user1", "user2"},
			"",
			[]string{"foo", "bar", "default"},
		},
		{
			"roles: list of string, multiple",
			"",
			nil,
			"rollen",
			[]string{"user1", "user2", "user2"},
			"",
			[]string{"foo", "bar", "default"},
		},
		{
			"roles: list of string, no additional 1",
			"",
			nil,
			"rollen",
			[]string{"admin", "user1"},
			"",
			[]string{"foo", "bar", "baz", "default"},
		},
		{
			"roles: list of string, no additional 2",
			"",
			nil,
			"rollen",
			[]string{"admin", "user2"},
			"",
			[]string{"foo", "bar", "baz", "default"},
		},
		{
			"roles: warn: boolean",
			"",
			nil,
			"roles",
			true,
			"invalid roles claim value type, ignoring claim, value: true",
			[]string{"default"},
		},
		{
			"roles: warn: number",
			"",
			nil,
			"roles",
			1.23,
			"invalid roles claim value type, ignoring claim, value: 1.23",
			[]string{"default"},
		},
		{
			"roles: warn: list of bool",
			"",
			nil,
			"roles",
			[]bool{true, false},
			"invalid roles claim value type, ignoring claim, value: []interface {}{true, false}",
			[]string{"default"},
		},
		{
			"roles: warn: list of number",
			"",
			nil,
			"roles",
			[]int{1, 2},
			"invalid roles claim value type, ignoring claim, value: []interface {}{1, 2}",
			[]string{"default"},
		},
		{
			"roles: warn: mixed list",
			"",
			nil,
			"roles",
			[]interface{}{"user1", 2},
			`invalid roles claim value type, ignoring claim, value: []interface {}{"user1", 2}`,
			[]string{"default"},
		},
		{
			"roles: warn: object",
			"",
			nil,
			"roles",
			map[string]interface{}{"foo": 1, "bar": 1},
			`invalid roles claim value type, ignoring claim, value: map[string]interface {}{"bar":1, "foo":1}`,
			[]string{"default"},
		},
		{
			"combi 1",
			"scope",
			"foo foo",
			"roles",
			[]string{"user2"},
			"",
			[]string{"foo", "bar", "default"},
		},
		{
			"combi 2",
			"scope",
			[]string{"foo", "bar"},
			"roles",
			"admin",
			"",
			[]string{"foo", "bar", "baz", "default"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()
			claims := jwt.MapClaims{}
			if tt.scopeClaim != "" && tt.scope != nil {
				claims[tt.scopeClaim] = tt.scope
			}
			if tt.rolesClaim != "" && tt.roles != nil {
				claims[tt.rolesClaim] = tt.roles
			}
			tok := jwt.NewWithClaims(signingMethod, claims)
			pubKeyBytes := []byte("mySecretK3y")
			token, tokenErr := tok.SignedString(pubKeyBytes)
			if tokenErr != nil {
				subT.Error(tokenErr)
			}

			source := ac.NewJWTSource("", "Authorization", nil)
			j, err := ac.NewJWT(&ac.JWTOptions{
				Algorithm:  algo.String(),
				Name:       "test_ac",
				ScopeClaim: tt.scopeClaim,
				RolesClaim: tt.rolesClaim,
				RolesMap:   rolesMap,
				Source:     source,
				Key:        pubKeyBytes,
			})
			if err != nil {
				subT.Fatal(err)
			}

			req := setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)
			req = req.WithContext(context.WithValue(context.Background(), request.LogEntry, log.WithContext(context.Background())))

			if err = j.Validate(req); err != nil {
				subT.Errorf("Unexpected error = %v", err)
				return
			}

			scopesList, ok := req.Context().Value(request.Scopes).([]string)
			if !ok {
				subT.Errorf("Expected scopes within request context")
			} else {
				if !reflect.DeepEqual(tt.expScopes, scopesList) {
					subT.Errorf("Scopes do not match, want: %#v, got: %#v", tt.expScopes, scopesList)
				}
			}

			entries := hook.AllEntries()
			if tt.expWarning == "" {
				if len(entries) > 0 {
					subT.Errorf("Expected no log messages, got: %d", len(entries))
				}
				return
			}
			if len(entries) != 1 {
				subT.Errorf("Expected one log message: got: %d", len(entries))
				return
			}
			entry := entries[0]
			if entry.Level != logrus.WarnLevel {
				subT.Errorf("Expected warning, got: %v", entry.Level)
				return
			}
			if entry.Message != tt.expWarning {
				subT.Errorf("Warning mismatch,\n\twant: %s,\n\tgot: %s", tt.expWarning, entry.Message)
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
			"missing both signature_algorithm/jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			  }
			}
			`,
			"signature_algorithm or jwks_url required",
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
			"jwt key: read error: required: configured attribute or file",
		},
		{
			"ok: signature_algorithm + key (default: header = Authorization)",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    key = "..."
			  }
			}
			`,
			"",
		},
		{
			"ok: signature_algorithm + key + header",
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
			"ok: signature_algorithm + key + cookie",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    cookie = "..."
			    key = "..."
			  }
			}
			`,
			"",
		},
		{
			"ok: signature_algorithm + key + token_value",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    token_value = env.TOKEN
			    key = "..."
			  }
			}
			`,
			"",
		},
		{
			"token_value + header",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    token_value = env.TOKEN
			    header = "..."
			    key = "..."
			  }
			}
			`,
			"token source is invalid",
		},
		{
			"token_value + cookie",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    token_value = env.TOKEN
			    cookie = "..."
			    key = "..."
			  }
			}
			`,
			"token source is invalid",
		},
		{
			"cookie + header",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    cookie = "..."
			    header = "..."
			    key = "..."
			  }
			}
			`,
			"token source is invalid",
		},
		{
			"ok: signature_algorithm + key_file",
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
			"ok: jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    jwks_url = "http://..."
			    header = "..."
			  }
			}
			`,
			"",
		},
		{
			"signature_algorithm + jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    jwks_url = "http://..."
			    header = "..."
			  }
			}
			`,
			"signature_algorithm cannot be used together with jwks_url",
		},
		{
			"key + jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    key = "..."
			    jwks_url = "http://..."
			    header = "..."
			  }
			}
			`,
			"key cannot be used together with jwks_url",
		},
		{
			"key_file + jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    key_file = "..."
			    jwks_url = "http://..."
			    header = "..."
			  }
			}
			`,
			"key_file cannot be used together with jwks_url",
		},
		{
			"backend reference, missing jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    backend = "foo"
			    header = "..."
			  }
			  backend "foo" {}
			}
			`,
			"backend not needed without jwks_url",
		},
		{
			"ok: jwks_url + backend reference",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    backend = "foo"
			    header = "..."
			    jwks_url = "http://..."
			  }
			  backend "foo" {}
			}
			`,
			"",
		},
	}

	log, _ := logrustest.NewNullLogger()

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if conf != nil {
				_, err = runtime.NewServerConfiguration(conf, log.WithContext(context.TODO()), nil)
			}

			var error = ""
			if err != nil {
				if _, ok := err.(errors.GoError); ok {
					error = err.(errors.GoError).LogError()
				} else {
					error = err.Error()
				}
			}

			if tt.error == "" && error == "" {
				return
			}

			expectedError := "configuration error: myac: " + tt.error

			if expectedError != error {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, expectedError, error)
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
