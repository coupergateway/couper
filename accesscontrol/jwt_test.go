package accesscontrol_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go/v4"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/internal/test"
)

func Test_JWT_NewJWT_RSA(t *testing.T) {
	helper := test.New(t)

	type fields struct {
		algorithm      string
		claims         map[string]interface{}
		claimsRequired []string
		pubKey         []byte
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
			{"missing key", fields{}, "missing key"},
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
			t.Run(fmt.Sprintf("%v / %s", signingMethod, tt.name), func(t *testing.T) {
				j, jerr := ac.NewJWT(&ac.JWTOptions{
					Algorithm:      tt.fields.algorithm,
					Claims:         tt.fields.claims,
					ClaimsRequired: tt.fields.claimsRequired,
					Name:           "test_ac",
					Key:            string(tt.fields.pubKey),
				})
				if jerr != nil {
					if tt.wantErr != jerr.Error() {
						t.Errorf("error: %v, want: %v", jerr, tt.wantErr)
					}
				} else if tt.wantErr != "" {
					t.Errorf("error expected: %v", tt.wantErr)
				}
				if tt.wantErr == "" {
					if j == nil {
						t.Errorf("JWT struct expected")
					}
				}
			})
		}
	}
}

func Test_JWT_Validate(t *testing.T) {
	type fields struct {
		algorithm      ac.Algorithm
		claims         map[string]interface{}
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

		algo := ac.NewAlgorithm(signingMethod.Alg())

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
			{"not configured", fields{}, httptest.NewRequest(http.MethodGet, "/", nil), true},
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
				claims: map[string]interface{}{
					"aud":     "peter",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				source:         ac.NewJWTSource("", "Authorization"),
				pubKey:         pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), false},
			{"src: header /w valid bearer & w/o claims", fields{
				algorithm: algo,
				claims: map[string]interface{}{
					"aud":  "peter",
					"cptn": "hook",
				},
				source: ac.NewJWTSource("", "Authorization"),
				pubKey: pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), true},
			{"src: header /w valid bearer & w/o required claims", fields{
				algorithm: algo,
				claims: map[string]interface{}{
					"aud": "peter",
				},
				claimsRequired: []string{"exp"},
				source:         ac.NewJWTSource("", "Authorization"),
				pubKey:         pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), true},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v_%s", signingMethod, tt.name), func(t *testing.T) {
				j, err := ac.NewJWT(&ac.JWTOptions{
					Algorithm:      tt.fields.algorithm.String(),
					Claims:         tt.fields.claims,
					ClaimsRequired: tt.fields.claimsRequired,
					Name:           "test_ac",
					Source:         tt.fields.source,
					Key:            string(tt.fields.pubKey),
				})

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
