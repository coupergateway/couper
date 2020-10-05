package accesscontrol

import (
	"bufio"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

var _ AccessControl = &BasicAuth{}

var (
	ErrorBasicAuthMissingCredentials = errors.New("missing credentials")
	ErrorBasicAuthNotConfigured      = errors.New("handler not configured")
	ErrorBasicAuthUnauthorized       = errors.New("unauthorized")
)

type BasicAuthError struct {
	error
	Realm string
}

const (
	InvalidLine uint8 = iota
	LineTooLong
	MalformedPassword
	MultipleUser
	NotSupported
)

type BasicAuthHTParseError struct {
	error
	code uint8
}

var basicAuthErrors = map[uint8]string{
	InvalidLine:       "invalidLine",
	LineTooLong:       "lineTooLong",
	MalformedPassword: "malformedPassword",
	MultipleUser:      "multipleUser",
	NotSupported:      "notSupported",
}

func (e *BasicAuthHTParseError) Error() string {
	return fmt.Sprintf("basic auth ht parse error: %s: %s", basicAuthErrors[e.code], e.error)
}

func (e BasicAuthError) Error() string {
	return e.error.Error()
}

// BasicAuth represents an AC-BasicAuth object
type BasicAuth struct {
	htFile htData
	name   string
	user   string
	pass   string
	realm  string
}

// NewBasicAuth creates a new AC-BasicAuth object
func NewBasicAuth(name, user, pass, file, realm string) (*BasicAuth, error) {
	ba := &BasicAuth{
		htFile: make(htData),
		name:   name,
		user:   user,
		pass:   pass,
		realm:  realm,
	}

	if file == "" {
		return ba, nil
	}

	fp, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	var lineNr int
	for scanner.Scan() {
		lineNr++
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if len(line) > 255 {
			return nil, &BasicAuthHTParseError{code: LineTooLong, error: fmt.Errorf("%s:%d", file, lineNr)}
		}

		up := strings.SplitN(line, ":", 2)
		if len(up) != 2 {
			return nil, &BasicAuthHTParseError{code: InvalidLine, error: fmt.Errorf("%s:%d", file, lineNr)}
		}

		if _, ok := ba.htFile[up[0]]; ok {
			return nil, &BasicAuthHTParseError{code: MultipleUser, error: fmt.Errorf("%s:%d: %q", file, lineNr, up[0])}
		}

		switch pwdType := getPwdType(up[1]); pwdType {
		case pwdTypeApr1:
			fallthrough
		case pwdTypeMD5:
			prefix := pwdPrefixApr1
			if pwdType == pwdTypeMD5 {
				prefix = pwdPrefixMD5
			}

			parts := strings.Split(strings.TrimPrefix(up[1], prefix), "$")
			if len(parts) != 2 {
				return nil, &BasicAuthHTParseError{code: MalformedPassword, error: fmt.Errorf("%s:%d: user %q", file, lineNr, up[0])}
			}

			ba.htFile[up[0]] = pwd{
				pwdOrig:   []byte(up[1]),
				pwdPrefix: prefix,
				pwdSalt:   parts[0],
				pwdType:   pwdType,
			}
		case pwdTypeBcrypt:
			ba.htFile[up[0]] = pwd{
				pwdOrig: []byte(up[1]),
				pwdType: pwdType,
			}
		default:
			return nil, &BasicAuthHTParseError{
				code:  NotSupported,
				error: fmt.Errorf("%s:%d: unknown password algorithm", file, lineNr),
			}
		}
	}

	err = scanner.Err()
	return ba, err
}

// Validate implements the AccessControl interface
func (ba *BasicAuth) Validate(req *http.Request) error {
	if ba == nil {
		return ErrorBasicAuthNotConfigured
	}

	user, pass, ok := req.BasicAuth()
	if !ok {
		return &BasicAuthError{error: ErrorBasicAuthMissingCredentials, Realm: ba.realm}
	}

	if ba.user == user {
		if subtle.ConstantTimeCompare([]byte(ba.pass), []byte(pass)) == 1 {
			return nil
		}
		return &BasicAuthError{error: ErrorBasicAuthUnauthorized, Realm: ba.realm}
	}

	if validateAccessData(user, pass, ba.htFile) {
		return nil
	}

	return &BasicAuthError{error: ErrorBasicAuthUnauthorized, Realm: ba.realm}
}
