package accesscontrol

import (
	"bufio"
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
)

var _ AccessControl = &BasicAuth{}

// BasicAuth represents an AC-BasicAuth object
type BasicAuth struct {
	htFile htData
	name   string
	user   string
	pass   string
}

// NewBasicAuth creates a new AC-BasicAuth object
func NewBasicAuth(name, user, pass, file string) (*BasicAuth, error) {
	ba := &BasicAuth{
		htFile: make(htData),
		name:   name,
		user:   user,
		pass:   pass,
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
			return nil, fmt.Errorf("parse error: line length exceeded: 255")
		}

		up := strings.SplitN(line, ":", 2)
		if len(up) != 2 {
			return nil, fmt.Errorf("parse error: invalid line: " + strconv.Itoa(lineNr))
		}

		username, password := up[0], up[1]

		if _, ok := ba.htFile[username]; ok {
			return nil, fmt.Errorf("multiple user: " + username)
		}

		switch pwdType := getPwdType(password); pwdType {
		case pwdTypeApr1:
			fallthrough
		case pwdTypeMD5:
			prefix := pwdPrefixApr1
			if pwdType == pwdTypeMD5 {
				prefix = pwdPrefixMD5
			}

			parts := strings.Split(strings.TrimPrefix(password, prefix), "$")
			if len(parts) != 2 {
				return nil, fmt.Errorf("parse error: malformed password for user: " + username)
			}

			ba.htFile[username] = pwd{
				pwdOrig:   []byte(password),
				pwdPrefix: prefix,
				pwdSalt:   parts[0],
				pwdType:   pwdType,
			}
		case pwdTypeBcrypt:
			ba.htFile[username] = pwd{
				pwdOrig: []byte(password),
				pwdType: pwdType,
			}
		default:
			return nil, fmt.Errorf("parse error: algorithm not supported")
		}
	}

	err = scanner.Err()
	return ba, err
}

// Validate implements the AccessControl interface
func (ba *BasicAuth) Validate(req *http.Request) error {
	if ba == nil {
		return errors.Configuration
	}

	user, pass, ok := req.BasicAuth()
	if !ok { // false is unspecific, determine if credentials are set
		const prefix = "Basic "
		if val := req.Header.Get("Authorization"); val == "" || !strings.HasPrefix(val, prefix) {
			return errors.BasicAuthCredentialsMissing.Message("credentials required")
		}
		return errors.BasicAuth.Message("reading authorization failed")
	}

	if ba.user == user {
		if ba.pass != "" {
			if subtle.ConstantTimeCompare([]byte(ba.pass), []byte(pass)) == 1 {
				return ba.withUsername(req, user)
			}
			return errors.BasicAuth.Message("credential mismatch")
		}

		if len(ba.htFile) == 0 {
			return errors.BasicAuth.Message("no password configured")
		}
	}

	if len(ba.htFile) > 0 {
		if validateAccessData(user, pass, ba.htFile) {
			return ba.withUsername(req, user)
		}
		return errors.BasicAuth.Message("file: credential mismatch")
	}

	return errors.BasicAuth.Message("credential mismatch")
}

func (ba *BasicAuth) withUsername(req *http.Request, user string) error {
	u := make(map[string]interface{})
	u["user"] = user

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[ba.name] = u

	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
}
