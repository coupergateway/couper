package writer

import (
	"net/http"
	"reflect"
	"testing"
)

func TestCookies_StripSecureCookies(t *testing.T) {
	header := http.Header{}

	header.Add(setCookieHeader, "name=1; HttpOnly")
	header.Add(setCookieHeader, "name=2; Path=path; Secure; HttpOnly")
	header.Add(setCookieHeader, "name=2; Path=path; Secure; HttpOnly;")
	header.Add(setCookieHeader, "name=3; Path=path; HttpOnly; Secure;")
	header.Add(setCookieHeader, "name=4; Path=path; HttpOnly; secure")
	header.Add(setCookieHeader, "name=secure; Path=path; HttpOnly")
	header.Add(setCookieHeader, "name=Secure; Path=path; HttpOnly;")

	stripSecureCookies(header)

	exp := http.Header{}
	exp.Add(setCookieHeader, "name=1; HttpOnly")
	exp.Add(setCookieHeader, "name=2; Path=path; HttpOnly")
	exp.Add(setCookieHeader, "name=2; Path=path; HttpOnly")
	exp.Add(setCookieHeader, "name=3; Path=path; HttpOnly")
	exp.Add(setCookieHeader, "name=4; Path=path; HttpOnly")
	exp.Add(setCookieHeader, "name=secure; Path=path; HttpOnly")
	exp.Add(setCookieHeader, "name=Secure; Path=path; HttpOnly;")

	if !reflect.DeepEqual(header, exp) {
		t.Errorf("Expected \n'%v', got: \n'%v'", exp, header)
	}
}
