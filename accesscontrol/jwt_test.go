package accesscontrol_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/coupergateway/couper/accesscontrol"
	acjwt "github.com/coupergateway/couper/accesscontrol/jwt"
	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/reader"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/config/runtime"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/test"
)

func Test_JWT_NewJWT_RSA(t *testing.T) {
	helper := test.New(t)
	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)
	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())
	memStore := cache.New(logger, tmpStoreCh)

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

				j, jerr := ac.NewJWT(&config.JWT{
					Claims:             tt.fields.claims,
					ClaimsRequired:     tt.fields.claimsRequired,
					Name:               "test_ac",
					SignatureAlgorithm: tt.fields.algorithm,
				}, nil, key, memStore)
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
	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)
	logger := log.WithContext(context.Background())
	memStore := cache.New(logger, tmpStoreCh)
	type fields struct {
		algorithm      acjwt.Algorithm
		bearer         bool
		claims         map[string]string
		claimsRequired []string
		cookie         string
		header         string
		tokenValue     hcl.Expression
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
			{"src: bearer /w no authorization header", fields{
				algorithm: algo,
				bearer:    true,
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), "jwt_token_missing"},
			{"src: bearer /w different auth-scheme", fields{
				algorithm: algo,
				bearer:    true,
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Basic qbqnb"), "jwt_token_missing"},
			{"src: bearer /w empty bearer", fields{
				algorithm: algo,
				bearer:    true,
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR"), "jwt_token_missing"},
			{"src: bearer /w valid bearer", fields{
				algorithm: algo,
				bearer:    true,
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), ""},
			{"src: bearer /w valid bearer & claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":     "peter",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				bearer:         true,
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), ""},
			{"src: bearer /w valid bearer & wrong audience", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":     "paul",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				bearer:         true,
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), "jwt_token_invalid"},
			{"src: bearer /w valid bearer & w/o claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud":  "peter",
					"cptn": "hook",
				},
				bearer: true,
				pubKey: pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), "jwt_token_invalid"},
			{"src: bearer /w valid bearer & w/o required claims", fields{
				algorithm: algo,
				claims: map[string]string{
					"aud": "peter",
				},
				claimsRequired: []string{"exp"},
				bearer:         true,
				pubKey:         pubKeyBytes,
			}, setContext(setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token)), "jwt_token_invalid"},
			{"src: header /w no authorization header", fields{
				algorithm: algo,
				header:    "Authorization",
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), "jwt_token_missing"},
			{"src: header /w different auth-scheme", fields{
				algorithm: algo,
				header:    "Authorization",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Basic qbqnb"), "jwt_token_missing"},
			{"src: header /w empty bearer", fields{
				algorithm: algo,
				header:    "Authorization",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR"), "jwt_token_missing"},
			{"src: header /w valid bearer", fields{
				algorithm: algo,
				header:    "Authorization",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), ""},
			{"src: cookie /w no cookie", fields{
				algorithm: algo,
				cookie:    "token",
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), "jwt_token_missing"},
			{"src: cookie /w empty cookie", fields{
				algorithm: algo,
				cookie:    "token",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", ""), "jwt_token_missing"},
			{"src: cookie /w valid cookie", fields{
				algorithm: algo,
				cookie:    "token",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", token), ""},
			{
				"token_value number",
				fields{
					algorithm:  algo,
					tokenValue: hcltest.MockExprLiteral(cty.NumberIntVal(42)),
					pubKey:     pubKeyBytes,
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
					tokenValue:     hcltest.MockExprLiteral(cty.StringVal(token)),
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
				j, err := ac.NewJWT(&config.JWT{
					SignatureAlgorithm: tt.fields.algorithm.String(),
					Claims:             hcl.StaticExpr(cty.ObjectVal(claimValMap), hcl.Range{}),
					ClaimsRequired:     tt.fields.claimsRequired,
					Name:               "test_ac",
					Bearer:             tt.fields.bearer,
					Cookie:             tt.fields.cookie,
					Header:             tt.fields.header,
					TokenValue:         tt.fields.tokenValue,
				}, nil, tt.fields.pubKey, memStore)
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

func Test_JWT_Validate_claims(t *testing.T) {
	log, _ := test.NewLogger()
	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)
	logger := log.WithContext(context.Background())
	memStore := cache.New(logger, tmpStoreCh)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"s": "abc",
		"i": 42,
		"f": 1.23,
		"b": true,
		"t": []float64{1.23, 3.21},
		"o": map[string]float64{"a": 0.0, "b": 1.1},
	})

	key := []byte("mySecretK3y")
	token, _ := tok.SignedString(key)

	type testCase struct {
		name        string
		req         *http.Request
		claims      map[string]interface{}
		wantErrKind string
	}

	for _, tc := range []testCase{
		{
			"all ok",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"s": "abc", "i": 42, "f": 1.23, "b": true, "t": []float64{1.23, 3.21}, "o": map[string]float64{"a": 0.0, "b": 1.1}},
			"",
		},
		{
			"wrong bool",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"b": false},
			"jwt_token_invalid",
		},
		{
			"wrong int",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"i": 0},
			"jwt_token_invalid",
		},
		{
			"wrong float",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"f": 3.21},
			"jwt_token_invalid",
		},
		{
			"wrong string",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"s": "asdf"},
			"jwt_token_invalid",
		},
		{
			"wrong tuple",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"t": []float64{2.34}},
			"jwt_token_invalid",
		},
		{
			"wrong object",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"o": map[string]float64{"c": 2.2}},
			"jwt_token_invalid",
		},
		{
			"missing expected claim",
			setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "Bearer "+token),
			map[string]interface{}{"expected": "str"},
			"jwt_token_invalid",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			claimValMap := make(map[string]cty.Value)
			for k, v := range tc.claims {
				switch val := v.(type) {
				case string:
					claimValMap[k] = cty.StringVal(val)
				case int:
					claimValMap[k] = cty.NumberIntVal(int64(val))
				case float64:
					claimValMap[k] = cty.NumberFloatVal(val)
				case bool:
					claimValMap[k] = cty.BoolVal(val)
				case []float64:
					var l []cty.Value
					for _, e := range val {
						l = append(l, cty.NumberFloatVal(e))
					}
					claimValMap[k] = cty.TupleVal(l)
				case map[string]float64:
					m := make(map[string]cty.Value)
					for mk, mv := range val {
						m[mk] = cty.NumberFloatVal(mv)
					}
					claimValMap[k] = cty.ObjectVal(m)
				default:
					subT.Fatal("must be one of the mapped types")
				}
			}
			j, err := ac.NewJWT(&config.JWT{
				SignatureAlgorithm: "HS256",
				Claims:             hcl.StaticExpr(cty.ObjectVal(claimValMap), hcl.Range{}),
				Bearer:             true,
			}, nil, key, memStore)
			if err != nil {
				subT.Error(err)
				return
			}

			tc.req = tc.req.WithContext(context.WithValue(context.Background(), request.LogEntry, log.WithContext(context.Background())))

			errKind := ""
			err = j.Validate(tc.req)
			if err != nil {
				cErr := err.(*errors.Error)
				errKind = cErr.Kinds()[0]
			}
			if errKind != tc.wantErrKind {
				subT.Errorf("Error want: %s, got: %s", tc.wantErrKind, errKind)
			}
		})
	}
}

func Test_JWT_DPoP(t *testing.T) {
	log, _ := test.NewLogger()
	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)
	logger := log.WithContext(context.Background())
	memStore := cache.New(logger, tmpStoreCh)
	h := test.New(t)

	signingMethod := jwt.SigningMethodRS256
	algo := acjwt.NewAlgorithm(signingMethod.Alg())

	privKeyPEMBytes := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC7VJTUt9Us8cKj
MzEfYyjiWA4R4/M2bS1GB4t7NXp98C3SC6dVMvDuictGeurT8jNbvJZHtCSuYEvu
NMoSfm76oqFvAp8Gy0iz5sxjZmSnXyCdPEovGhLa0VzMaQ8s+CLOyS56YyCFGeJZ
qgtzJ6GR3eqoYSW9b9UMvkBpZODSctWSNGj3P7jRFDO5VoTwCQAWbFnOjDfH5Ulg
p2PKSQnSJP3AJLQNFNe7br1XbrhV//eO+t51mIpGSDCUv3E0DDFcWDTH9cXDTTlR
ZVEiR2BwpZOOkE/Z0/BVnhZYL71oZV34bKfWjQIt6V/isSMahdsAASACp4ZTGtwi
VuNd9tybAgMBAAECggEBAKTmjaS6tkK8BlPXClTQ2vpz/N6uxDeS35mXpqasqskV
laAidgg/sWqpjXDbXr93otIMLlWsM+X0CqMDgSXKejLS2jx4GDjI1ZTXg++0AMJ8
sJ74pWzVDOfmCEQ/7wXs3+cbnXhKriO8Z036q92Qc1+N87SI38nkGa0ABH9CN83H
mQqt4fB7UdHzuIRe/me2PGhIq5ZBzj6h3BpoPGzEP+x3l9YmK8t/1cN0pqI+dQwY
dgfGjackLu/2qH80MCF7IyQaseZUOJyKrCLtSD/Iixv/hzDEUPfOCjFDgTpzf3cw
ta8+oE4wHCo1iI1/4TlPkwmXx4qSXtmw4aQPz7IDQvECgYEA8KNThCO2gsC2I9PQ
DM/8Cw0O983WCDY+oi+7JPiNAJwv5DYBqEZB1QYdj06YD16XlC/HAZMsMku1na2T
N0driwenQQWzoev3g2S7gRDoS/FCJSI3jJ+kjgtaA7Qmzlgk1TxODN+G1H91HW7t
0l7VnL27IWyYo2qRRK3jzxqUiPUCgYEAx0oQs2reBQGMVZnApD1jeq7n4MvNLcPv
t8b/eU9iUv6Y4Mj0Suo/AU8lYZXm8ubbqAlwz2VSVunD2tOplHyMUrtCtObAfVDU
AhCndKaA9gApgfb3xw1IKbuQ1u4IF1FJl3VtumfQn//LiH1B3rXhcdyo3/vIttEk
48RakUKClU8CgYEAzV7W3COOlDDcQd935DdtKBFRAPRPAlspQUnzMi5eSHMD/ISL
DY5IiQHbIH83D4bvXq0X7qQoSBSNP7Dvv3HYuqMhf0DaegrlBuJllFVVq9qPVRnK
xt1Il2HgxOBvbhOT+9in1BzA+YJ99UzC85O0Qz06A+CmtHEy4aZ2kj5hHjECgYEA
mNS4+A8Fkss8Js1RieK2LniBxMgmYml3pfVLKGnzmng7H2+cwPLhPIzIuwytXywh
2bzbsYEfYx3EoEVgMEpPhoarQnYPukrJO4gwE2o5Te6T5mJSZGlQJQj9q4ZB2Dfz
et6INsK0oG8XVGXSpQvQh3RUYekCZQkBBFcpqWpbIEsCgYAnM3DQf3FJoSnXaMhr
VBIovic5l0xFkEHskAjFTevO86Fsz1C2aSeRKSqGFoOQ0tmJzBEs1R6KqnHInicD
TQrKhArgLXX4v3CddjfTRJkFWDbE/CkvKZNOrcf1nhaGCPspRJj2KUkj1Fhl9Cnc
dn/RsYEONbwQSjIfMPkvxF+8HQ==
-----END PRIVATE KEY-----`)
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyPEMBytes)
	h.Must(err)
	pubKeyBytes := []byte(`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu1SU1LfVLPHCozMxH2Mo
4lgOEePzNm0tRgeLezV6ffAt0gunVTLw7onLRnrq0/IzW7yWR7QkrmBL7jTKEn5u
+qKhbwKfBstIs+bMY2Zkp18gnTxKLxoS2tFczGkPLPgizskuemMghRniWaoLcyeh
kd3qqGElvW/VDL5AaWTg0nLVkjRo9z+40RQzuVaE8AkAFmxZzow3x+VJYKdjykkJ
0iT9wCS0DRTXu269V264Vf/3jvredZiKRkgwlL9xNAwxXFg0x/XFw005UWVRIkdg
cKWTjpBP2dPwVZ4WWC+9aGVd+Gyn1o0CLelf4rEjGoXbAAEgAqeGUxrcIlbjXfbc
mwIDAQAB
-----END PUBLIC KEY-----`)
	jwk := test.RSAPubKeyToJWK(privKey.PublicKey)
	jkt := ac.JwkToJKT(jwk)

	jwtAC, err := ac.NewJWT(&config.JWT{
		Dpop:               true,
		SignatureAlgorithm: algo.String(),
	}, nil, pubKeyBytes, memStore)
	h.Must(err)

	type testCase struct {
		name       string
		authScheme string
		setProof   bool
		expErrMsg  string
	}

	for _, tc := range []testCase{
		{
			"ok", "DPoP", true, "",
		},
		{
			"missing DPoP proof", "DPoP", false, "access control error: missing DPoP request header field",
		},
		{
			"bearer token", "Bearer", false, `access control error: auth scheme "DPoP" required in authorization header`,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)
			req = req.WithContext(context.WithValue(context.Background(), request.LogEntry, log.WithContext(context.Background())))

			accessTokenClaims := jwt.MapClaims{
				"cnf": map[string]interface{}{
					"jkt": jkt,
				},
			}
			at := jwt.NewWithClaims(signingMethod, accessTokenClaims)
			accessToken, err := at.SignedString(privKey)
			helper.Must(err)
			hash := sha256.Sum256([]byte(accessToken))
			ath := base64.RawURLEncoding.EncodeToString(hash[:])
			req.Header.Set("Authorization", tc.authScheme+" "+accessToken)

			if tc.setProof {
				proofClaims := jwt.MapClaims{
					"ath": ath,
					"htm": req.Method,
					"htu": req.URL.String(),
					"iat": time.Now().Unix(),
					"jti": "some_id",
				}
				p := jwt.NewWithClaims(signingMethod, proofClaims)
				p.Header["jwk"] = jwk
				p.Header["typ"] = ac.DpopTyp
				proof, err := p.SignedString(privKey)
				helper.Must(err)
				req.Header.Set("DPoP", proof)
			}

			err = jwtAC.Validate(req)
			if err != nil {
				msg := err.Error()
				if _, ok := err.(errors.GoError); ok {
					msg = err.(errors.GoError).LogError()
				}
				if tc.expErrMsg == "" {
					subT.Errorf("expected no error, but got: %q", msg)
				} else if msg != tc.expErrMsg {
					subT.Errorf("expected error message: %q, got: %q", tc.expErrMsg, msg)
				}
			} else if tc.expErrMsg != "" {
				subT.Errorf("expected err: %q, got no error", tc.expErrMsg)
			}
		})
	}
}

func Test_JWT_yields_permissions(t *testing.T) {
	log, hook := test.NewLogger()
	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)
	logger := log.WithContext(context.Background())
	memStore := cache.New(logger, tmpStoreCh)
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

			j, err := ac.NewJWT(&config.JWT{
				Name:               "test_ac",
				PermissionsClaim:   tt.permissionsClaim,
				PermissionsMap:     permissionsMap,
				RolesClaim:         tt.rolesClaim,
				RolesMap:           rolesMap,
				SignatureAlgorithm: algo.String(),
			}, nil, pubKeyBytes, memStore)
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
			"ok: signature_algorithm + key (default: bearer = true)",
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
			"ok: signature_algorithm + key + bearer",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    key = "..."
			    bearer = true
			  }
			}
			`,
			"",
		},
		{
			"ok: signature_algorithm + key + dpop",
			`
			server "test" {}
			definitions {
			  jwt "myac" {
			    signature_algorithm = "HS256"
			    key = "..."
			    beta_dpop = true
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

func Test_JWT_Validate_Concurrency(t *testing.T) {
	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)
	log, _ := test.NewLogger()
	ctx := context.Background()
	logger := log.WithContext(ctx)
	memStore := cache.New(logger, tmpStoreCh)

	claimValMap := map[string]cty.Value{
		"aud": cty.StringVal("my_audience"),
		"iss": cty.StringVal("my_issuer"),
	}
	key := []byte("asdf")
	j, err := ac.NewJWT(&config.JWT{
		SignatureAlgorithm: "HS256",
		Claims:             hcl.StaticExpr(cty.ObjectVal(claimValMap), hcl.Range{}),
		Name:               "test_ac",
		Bearer:             true,
	}, nil, key, memStore)
	if err != nil {
		t.Error(err)
		return
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "my_audience",
		"iss": "my_issuer",
	})
	token, err := tok.SignedString(key)
	if err != nil {
		t.Error(err)
		return
	}
	validate := func(wg *sync.WaitGroup) {
		defer wg.Done()
		req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil)
		if err != nil {
			t.Error(err)
			return
		}
		req = req.WithContext(context.WithValue(ctx, request.LogEntry, log.WithContext(ctx)))
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		_ = j.Validate(req)
	}
	var wg sync.WaitGroup
	wg.Add(3)
	go validate(&wg)
	go validate(&wg)
	go validate(&wg)
	wg.Wait()
}
