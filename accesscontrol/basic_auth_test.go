package accesscontrol_test

import (
	b64 "encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	ac "github.com/avenga/couper/accesscontrol"
)

func Test_NewBasicAuth(t *testing.T) {
	ba, err := ac.NewBasicAuth("name", "user", "pass", "", "Basic")
	if ba == nil || err != nil {
		t.Errorf("Got unexpected error: '%s'", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd", "Basic")
	if ba == nil || err != nil {
		t.Errorf("Got unexpected error: '%s'", err)
	}

	ba, err = ac.NewBasicAuth("name", "john", "pass", "testdata/htpasswd", "Basic")
	if ba == nil || err != nil {
		t.Errorf("Got unexpected error: '%s'", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "file", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "no such file or directory") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_invalid", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Invalid line") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_too_long", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Too long line") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_malformed", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Malformed") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_multi", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Multiple user") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_unsupported", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Unsupported password algorithm") {
		t.Errorf("Got unexpected error: %s", err)
	}
}

func Test_Validate(t *testing.T) {
	var ba *ac.BasicAuth
	var req *http.Request = &http.Request{}

	if err := ba.Validate(req); err != ac.ErrorBasicAuthNotConfigured {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header = http.Header{}
	ba, _ = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd", "Basic")
	if ba == nil {
		t.Fatal("Got unexpected error")
	}

	var ebau *ac.ErrorBAUnauthorized

	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Foo")
	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Basic X")
	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte("usr:pwd:foo")))
	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte("user:pass")))
	if err := ba.Validate(req); err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "bAsIc   "+b64.StdEncoding.EncodeToString([]byte("user:pass")))
	if err := ba.Validate(req); err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Asdfg "+b64.StdEncoding.EncodeToString([]byte("user:bass")))
	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte("user:bass")))
	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte("john:my-pass")))
	if err := ba.Validate(req); err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	req.Header.Set("Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte("john:my-bass")))
	if err := ba.Validate(req); !errors.As(err, &ebau) {
		t.Errorf("Got unexpected error: %s", err)
	}
}
