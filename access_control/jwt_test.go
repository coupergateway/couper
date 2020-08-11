package access_control_test

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

	ac "go.avenga.cloud/couper/gateway/access_control"
)

func TestJWT_Validate(t *testing.T) {
	type fields struct {
		algorithm ac.Algorithm
		source    ac.Source
		sourceKey string
		pubKey    []byte
	}

	for _, signingMethod := range []jwt.SigningMethod{jwt.SigningMethodRS256, jwt.SigningMethodHS256} {

		pubKeyBytes, privKey := newRSAKeyPair()

		tok := jwt.New(signingMethod)
		tok.Header["test123"] = "value123"
		var token string
		var tokenErr error

		var algo ac.Algorithm
		switch signingMethod {
		case jwt.SigningMethodHS256:
			token, tokenErr = tok.SignedString(pubKeyBytes)
			algo = ac.AlgorithmHMAC
		case jwt.SigningMethodRS256:
			token, tokenErr = tok.SignedString(privKey)
			algo = ac.AlgorithmRSA
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
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v_%s", signingMethod, tt.name), func(t *testing.T) {
				j, err := ac.NewJWT(tt.fields.algorithm.String(), ac.Claims{}, tt.fields.source, tt.fields.sourceKey, tt.fields.pubKey)

				if err = j.Validate(tt.req); (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
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
