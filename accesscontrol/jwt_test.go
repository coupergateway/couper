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

	ac "github.com/avenga/couper/accesscontrol"
)

func TestJWT_Validate(t *testing.T) {
	type fields struct {
		algorithm      ac.Algorithm
		claims         ac.Claims
		claimsRequired []string
		source         ac.Source
		sourceKey      string
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

		if strings.HasPrefix(signingMethod.Alg(), "HS") {
			token, tokenErr = tok.SignedString(pubKeyBytes)
		} else if strings.HasPrefix(signingMethod.Alg(), "RS") {
			token, tokenErr = tok.SignedString(privKey)
		}
		algo := ac.NewAlgorithm(signingMethod.Alg())

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
				source:    ac.Header,
				sourceKey: "Authorization",
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), true},
			{"src: header /w valid bearer", fields{
				algorithm: algo,
				source:    ac.Header,
				sourceKey: "Authorization",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), false},
			{"src: header /w no cookie", fields{
				algorithm: algo,
				source:    ac.Cookie,
				sourceKey: "token",
				pubKey:    pubKeyBytes,
			}, httptest.NewRequest(http.MethodGet, "/", nil), true},
			{"src: header /w empty cookie", fields{
				algorithm: algo,
				source:    ac.Cookie,
				sourceKey: "token",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", ""), true},
			{"src: header /w valid cookie", fields{
				algorithm: algo,
				source:    ac.Cookie,
				sourceKey: "token",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "token", token), false},
			{"src: header /w valid bearer & claims", fields{
				algorithm: algo,
				claims: ac.Claims{
					"aud":     "peter",
					"test123": "value123",
				},
				claimsRequired: []string{"aud"},
				source:         ac.Header,
				sourceKey:      "Authorization",
				pubKey:         pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), false},
			{"src: header /w valid bearer & w/o claims", fields{
				algorithm: algo,
				claims: ac.Claims{
					"aud":  "peter",
					"cptn": "hook",
				},
				source:    ac.Header,
				sourceKey: "Authorization",
				pubKey:    pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), true},
			{"src: header /w valid bearer & w/o required claims", fields{
				algorithm: algo,
				claims: ac.Claims{
					"aud": "peter",
				},
				claimsRequired: []string{"exp"},
				source:         ac.Header,
				sourceKey:      "Authorization",
				pubKey:         pubKeyBytes,
			}, setCookieAndHeader(httptest.NewRequest(http.MethodGet, "/", nil), "Authorization", "BeAreR "+token), true},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v_%s", signingMethod, tt.name), func(t *testing.T) {
				j, err := ac.NewJWT(tt.fields.algorithm.String(), "test_ac", tt.fields.claims, tt.fields.claimsRequired, tt.fields.source, tt.fields.sourceKey, tt.fields.pubKey)

				if err = j.Validate(tt.req); (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}

				if !tt.wantErr && tt.fields.claims != nil {
					acMap := tt.req.Context().Value(ac.ContextAccessControlKey).(map[string]interface{})
					if claims, ok := acMap["test_ac"]; !ok {
						t.Errorf("Expected a configured access control name within request context")
					} else {
						claimsMap := claims.(ac.Claims)
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
