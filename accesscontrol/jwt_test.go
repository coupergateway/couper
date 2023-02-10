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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/avenga/couper/accesscontrol"
	acjwt "github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/cache"
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
					Source:         ac.NewJWTSource("", "Authorization", nil),
				}, key)
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
			name        string
			fields      fields
			req         *http.Request
			wantErrKind string
		}{
			{"src: header /w no authorization header", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization", nil),
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), "jwt_token_missing"},
			{"src: header /w different auth-scheme", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Basic qbqnb"), "jwt_token_missing"},
			{"src: header /w empty bearer", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR"), "jwt_token_missing"},
			{"src: header /w valid bearer", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("", "Authorization", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), ""},
			{"src: header /w no cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", "", nil),
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), "jwt_token_missing"},
			{"src: header /w empty cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", "", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", ""), "jwt_token_missing"},
			{"src: header /w valid cookie", fields{
				algorithm: algo,
				source:    ac.NewJWTSource("token", "", nil),
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", token), ""},
			{"src: header /w valid bearer & claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":     "peter",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				source:         ac.NewJWTSource("", "Authorization", nil),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), ""},
			{"src: header /w valid bearer & wrong audience", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":     "paul",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				source:         ac.NewJWTSource("", "Authorization", nil),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), "jwt_token_invalid"},
			{"src: header /w valid bearer & w/o claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":  "peter",
					"cptn": "hook",
				},
				source: ac.NewJWTSource("", "Authorization", nil),
				pubKey: pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), "jwt_token_invalid"},
			{"src: header /w valid bearer & w/o required claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud": "peter",
				},
				claimsRequired: []string{"exp"},
				source:         ac.NewJWTSource("", "Authorization", nil),
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), "jwt_token_invalid"},
			{
				"token_value number",
				fields{
					algorithm: algo,
					source:    ac.NewJWTSource("", "", hcltest.MockExprLiteral(cty.NumberIntVal(42))),
					pubKey:    pubKeyBytes,
				},
				setContext(httptest.NewRequest(http.MethodGet, "/", nil)),
				"jwt_token_invalid",
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
				"",
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
				}, tt.fields.pubKey)
				if err != nil {
					subT.Error(err)
					return
				}

				tt.req = tt.req.WithContext(context.WithValue(context.Background(), request.LogEntry, log.WithContext(context.Background())))

				errKind := ""
				err = j.Validate(tt.req)
				if err != nil {
					cErr := err.(*errors.Error)
					errKind = cErr.Kinds()[0]
				}
				if errKind != tt.wantErrKind {
					subT.Errorf("Validate() error kind does not match; want: %q, got: %q", tt.wantErrKind, errKind)
				}

				if tt.wantErrKind == "" && tt.fields.claims != nil {
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

func Test_JWT_yields_permissions(t *testing.T) {
	log, hook := test.NewLogger()
	signingMethod := jwt.SigningMethodHS256
	algo := acjwt.NewAlgorithm(signingMethod.Alg())

	rolesMap := map[string][]string{
		"admin": {"foo", "bar", "baz"},
		"user1": {"foo"},
		"user2": {"bar"},
		"*":     {"default"},
	}
	permissionsMap := map[string][]string{
		"baz": {"blubb"},
	}
	var noGrantedPermissions []string

	tests := []struct {
		name             string
		permissionsClaim string
		permissionsValue interface{}
		rolesClaim       string
		rolesValue       interface{}
		expWarning       string
		expGrantedPerms  []string
	}{
		{
			"no permissions, no roles",
			"scp",
			nil,
			"roles",
			nil,
			"",
			noGrantedPermissions,
		},
		{
			"permissions: space-separated list",
			"scp",
			"foo bar",
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"permissions: space-separated list, multiple",
			"scp",
			"foo bar foo",
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"permissions: list of string",
			"scoop",
			[]string{"foo", "bar"},
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"permissions: list of string, multiple",
			"scoop",
			[]string{"foo", "bar", "bar"},
			"",
			nil,
			"",
			[]string{"foo", "bar"},
		},
		{
			"permissions: warn: boolean",
			"scope",
			true,
			"",
			nil,
			"invalid permissions claim value type, ignoring claim, value true",
			noGrantedPermissions,
		},
		{
			"permissions: warn: number",
			"scope",
			1.23,
			"",
			nil,
			"invalid permissions claim value type, ignoring claim, value 1.23",
			noGrantedPermissions,
		},
		{
			"permissions: warn: list of bool",
			"scope",
			[]bool{true, false},
			"",
			nil,
			"invalid permissions claim value type, ignoring claim, value []interface {}{true, false}",
			noGrantedPermissions,
		},
		{
			"permissions: warn: list of number",
			"scope",
			[]int{1, 2},
			"",
			nil,
			"invalid permissions claim value type, ignoring claim, value []interface {}{1, 2}",
			noGrantedPermissions,
		},
		{
			"permissions: warn: mixed list",
			"scope",
			[]interface{}{"eins", 2},
			"",
			nil,
			`invalid permissions claim value type, ignoring claim, value []interface {}{"eins", 2}`,
			noGrantedPermissions,
		},
		{
			"permissions: warn: object",
			"scope",
			map[string]interface{}{"foo": 1, "bar": 1},
			"",
			nil,
			`invalid permissions claim value type, ignoring claim, value map[string]interface {}{"bar":1, "foo":1}`,
			noGrantedPermissions,
		},
		{
			"roles: single string, permission mapped",
			"",
			nil,
			"roles",
			"admin",
			"",
			[]string{"foo", "bar", "baz", "default", "blubb"},
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
			"roles: list of string, no additional 1, permission mapped",
			"",
			nil,
			"rollen",
			[]string{"admin", "user1"},
			"",
			[]string{"foo", "bar", "baz", "default", "blubb"},
		},
		{
			"roles: list of string, no additional 2, permission mapped",
			"",
			nil,
			"rollen",
			[]string{"admin", "user2"},
			"",
			[]string{"foo", "bar", "baz", "default", "blubb"},
		},
		{
			"roles: warn: boolean",
			"",
			nil,
			"roles",
			true,
			"invalid roles claim value type, ignoring claim, value true",
			[]string{"default"},
		},
		{
			"roles: warn: number",
			"",
			nil,
			"roles",
			1.23,
			"invalid roles claim value type, ignoring claim, value 1.23",
			[]string{"default"},
		},
		{
			"roles: warn: list of bool",
			"",
			nil,
			"roles",
			[]bool{true, false},
			"invalid roles claim value type, ignoring claim, value []interface {}{true, false}",
			[]string{"default"},
		},
		{
			"roles: warn: list of number",
			"",
			nil,
			"roles",
			[]int{1, 2},
			"invalid roles claim value type, ignoring claim, value []interface {}{1, 2}",
			[]string{"default"},
		},
		{
			"roles: warn: mixed list",
			"",
			nil,
			"roles",
			[]interface{}{"user1", 2},
			`invalid roles claim value type, ignoring claim, value []interface {}{"user1", 2}`,
			[]string{"default"},
		},
		{
			"roles: warn: object",
			"",
			nil,
			"roles",
			map[string]interface{}{"foo": 1, "bar": 1},
			`invalid roles claim value type, ignoring claim, value map[string]interface {}{"bar":1, "foo":1}`,
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
			"combi 2, permission mapped",
			"scope",
			[]string{"foo", "bar"},
			"roles",
			"admin",
			"",
			[]string{"foo", "bar", "baz", "default", "blubb"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()
			claims := jwt.MapClaims{}
			if tt.permissionsClaim != "" && tt.permissionsValue != nil {
				claims[tt.permissionsClaim] = tt.permissionsValue
			}
			if tt.rolesClaim != "" && tt.rolesValue != nil {
				claims[tt.rolesClaim] = tt.rolesValue
			}
			tok := jwt.NewWithClaims(signingMethod, claims)
			pubKeyBytes := []byte("mySecretK3y")
			token, tokenErr := tok.SignedString(pubKeyBytes)
			if tokenErr != nil {
				subT.Error(tokenErr)
			}

			source := ac.NewJWTSource("", "Authorization", nil)
			j, err := ac.NewJWT(&ac.JWTOptions{
				Algorithm:        algo.String(),
				Name:             "test_ac",
				PermissionsClaim: tt.permissionsClaim,
				PermissionsMap:   permissionsMap,
				RolesClaim:       tt.rolesClaim,
				RolesMap:         rolesMap,
				Source:           source,
			}, pubKeyBytes)
			if err != nil {
				subT.Fatal(err)
			}

			req := setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)
			req = req.WithContext(context.WithValue(context.Background(), request.LogEntry, log.WithContext(context.Background())))

			if err = j.Validate(req); err != nil {
				subT.Errorf("Unexpected error = %v", err)
				return
			}

			grantedPermissionsList, ok := req.Context().Value(request.GrantedPermissions).([]string)
			if !ok {
				subT.Errorf("Expected granted permissions within request context")
			} else {
				if !reflect.DeepEqual(tt.expGrantedPerms, grantedPermissionsList) {
					subT.Errorf("Granted permissions do not match, want: %#v, got: %#v", tt.expGrantedPerms, grantedPermissionsList)
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

	const backendURL = "http://blackhole.webpagetest.org/"

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
			"configuration error: myac: signature_algorithm or jwks_url attribute required",
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
			"configuration error: myac: jwt key: read error: required: configured attribute or file",
		},
		{
			"signature_algorithm, both key and key_file",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    header = "..."
			    key = "..."
			    key_file = "testdata/secret.txt"
			  }
			}
			`,
			"configuration error: myac: jwt key: read error: configured attribute and file",
		},
		{
			"signature_algorithm, both roles_map and roles_map_file",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    header = "..."
			    key = "..."
			    roles_map = {}
			    roles_map_file = "testdata/map.json"
			  }
			}
			`,
			"configuration error: myac: jwt roles map: read error: configured attribute and file",
		},
		{
			"signature_algorithm, roles_map_file not found",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    header = "..."
			    key = "..."
			    roles_map_file = "file_not_found"
			  }
			}
			`,
			"configuration error: myac: roles map: read error: open .*/testdata/file_not_found: no such file or directory",
		},
		{
			"signature_algorithm, both permissions_map and permissions_map_file",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    header = "..."
			    key = "..."
			    permissions_map = {}
			    permissions_map_file = "testdata/map.json"
			  }
			}
			`,
			"configuration error: myac: jwt permissions map: read error: configured attribute and file",
		},
		{
			"signature_algorithm, permissions_map_file not found",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    header = "..."
			    key = "..."
			    permissions_map_file = "file_not_found"
			  }
			}
			`,
			"configuration error: myac: permissions map: read error: open .*/accesscontrol/file_not_found: no such file or directory",
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
			"configuration error: myac: token source is invalid",
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
			"configuration error: myac: token source is invalid",
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
			"configuration error: myac: token source is invalid",
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
			    jwks_url = "file:jwk/testdata/jwks.json",
			    header = "..."
			  }
			}
			`,
			"",
		},
		{
			"jwks_url file not found",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    jwks_url = "file:file_not_found",
			    header = "..."
			  }
			}
			`,
			"configuration error: myac: jwks_url: read error: open .*/accesscontrol/file_not_found: no such file or directory",
		},
		{
			"signature_algorithm + jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    jwks_url = "` + backendURL + `"
			    header = "..."
			  }
			}
			`,
			"configuration error: myac: signature_algorithm cannot be used together with jwks_url",
		},
		{
			"key + jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    key = "..."
			    jwks_url = "` + backendURL + `"
			    header = "..."
			  }
			}
			`,
			"configuration error: myac: key cannot be used together with jwks_url",
		},
		{
			"key_file + jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    key_file = "..."
			    jwks_url = "` + backendURL + `"
			    header = "..."
			  }
			}
			`,
			"configuration error: myac: key_file cannot be used together with jwks_url",
		},
		{
			"backend reference, missing jwks_url",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    backend = "foo"
			    header = "..."
				signature_algorithm = "asdf"
			  }
			  backend "foo" {}
			}
			`,
			"configuration error: myac: backend is obsolete without jwks_url attribute",
		},
	}

	log, hook := test.NewLogger()
	helper := test.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()

			conf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if conf != nil {
				tmpStoreCh := make(chan struct{})
				defer close(tmpStoreCh)

				ctx, cancel := context.WithCancel(conf.Context)
				conf.Context = ctx
				defer cancel()

				logger := log.WithContext(ctx)

				_, err = runtime.NewServerConfiguration(conf, logger, cache.New(logger, tmpStoreCh))
			}

			var errMsg, expectedError string
			if err != nil {
				if _, ok := err.(errors.GoError); ok {
					errMsg = err.(errors.GoError).LogError()
				} else {
					errMsg = err.Error()
				}
			}

			if tt.error == "" && errMsg == "" {
				return
			}

			time.Sleep(time.Second / 2) // sync routine start

			for _, e := range hook.AllEntries() {
				if e.Level != logrus.ErrorLevel {
					continue
				}
				errMsg = e.Message
				break
			}

			re, err := regexp.Compile(expectedError)
			helper.Must(err)
			if !re.MatchString(errMsg) {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, expectedError, errMsg)
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
