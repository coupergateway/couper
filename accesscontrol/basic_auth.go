package accesscontrol

import (
	"bufio"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	errors "github.com/avenga/couper/errors/accesscontrol/basic_auth"
)

var _ AccessControl = &BasicAuth{}

type BasicAuthError struct {
	error
	Realm string
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
			return nil, errors.ParseErrorLineLengthExceeded
		}

		up := strings.SplitN(line, ":", 2)
		if len(up) != 2 {
			return nil, errors.ParseErrorLineInvalid
		}

		if _, ok := ba.htFile[up[0]]; ok {
			return nil, errors.ParseErrorMultipleUser
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
				return nil, errors.ParseErrorMalformedPassword
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
			return nil, errors.ParseErrorAlgorithmNotSupported
		}
	}

	err = scanner.Err()
	return ba, err
}

// Validate implements the AccessControl interface
func (ba *BasicAuth) Validate(req *http.Request) error {
	if ba == nil {
		return errors.NotConfigured
	}

	if ba.pass == "" {
		return &BasicAuthError{error: errors.Unauthorized, Realm: ba.realm}
	}

	user, pass, ok := req.BasicAuth()
	if !ok {
		return &BasicAuthError{error: errors.MissingCredentials, Realm: ba.realm}
	}

	if ba.user == user {
		if subtle.ConstantTimeCompare([]byte(ba.pass), []byte(pass)) == 1 {
			return nil
		}
		return &BasicAuthError{error: errors.Unauthorized, Realm: ba.realm}
	}

	if validateAccessData(user, pass, ba.htFile) {
		return nil
	}

	return &BasicAuthError{error: errors.Unauthorized, Realm: ba.realm}
}
