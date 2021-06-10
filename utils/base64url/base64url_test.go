package base64url_test

import (
	"testing"

	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/utils/base64url"
)

func TestUtils_Base64url_EncodeDecode(t *testing.T) {
	helper := test.New(t)

	tests := []struct {
		orig, encoded string
	}{
		// https://tools.ietf.org/html/rfc7515#appendix-C
		{string([]byte{3, 236, 255, 224, 193}), "A-z_4ME"},
		// https://tools.ietf.org/html/rfc7515#appendix-A.1.1
		{`{"typ":"JWT",` + "\r\n " + `"alg":"HS256"}`, "eyJ0eXAiOiJKV1QiLA0KICJhbGciOiJIUzI1NiJ9"},
		// https://tools.ietf.org/html/rfc7515#appendix-A.4.1
		{"Payload", "UGF5bG9hZA"},
		// https://tools.ietf.org/html/rfc7515#appendix-A.5
		{`{"alg":"none"}`, "eyJhbGciOiJub25lIn0"},
		// https://tools.ietf.org/html/rfc7515#appendix-A.6.1
		{`{"alg":"RS256"}`, "eyJhbGciOiJSUzI1NiJ9"},
		{`{"alg":"ES256"}`, "eyJhbGciOiJFUzI1NiJ9"},
		// beliebige string-laenge (zum selbst-extrapolieren)
		{"x", "eA"},
		{"xx", "eHg"},
		{"xxx", "eHh4"},
		{"xxxx", "eHh4eA"},
		{"xxxxx", "eHh4eHg"},
		{"xxxxxx", "eHh4eHh4"},
	}
	for _, tt := range tests {
		t.Run(tt.orig, func(t *testing.T) {
			encoded := base64url.Encode([]byte(tt.orig))
			if encoded != tt.encoded {
				t.Errorf("Encode() got %v, want %v", encoded, tt.encoded)
			}
			decoded, err := base64url.Decode(encoded)
			if err != nil {
				helper.Must(err)
			}
			if string(decoded) != tt.orig {
				t.Errorf("Decode() got %v, want %v", string(decoded), tt.orig)
			}
		})
	}
}

func TestUtils_Base64url_DecodeInvalidLength(t *testing.T) {
	decoded, err := base64url.Decode("X")
	want := "invalid base64url string"
	if err == nil {
		t.Errorf("Expected Decode() to fail, got %v", decoded)
	} else if err.Error() != want {
		t.Errorf("Decode() error got %v, want %v", err.Error(), want)
	}
}

func TestUtils_Base64url_DecodeInvalidCharacters(t *testing.T) {
	decoded, err := base64url.Decode("??")
	want := "illegal base64 data at input byte 0"
	if err == nil {
		t.Errorf("Expected Decode() to fail, got %v", string(decoded))
	} else if err.Error() != want {
		t.Errorf("Decode() error got %v, want %v", err.Error(), want)
	}
}
