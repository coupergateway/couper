package accesscontrol

import (
	"bufio"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
)

var _ AccessControl = &BasicAuth{}

const authHeader = "Authorization"

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
	}

	if file != "" {
		fp, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer fp.Close()

		scanner := bufio.NewScanner(fp)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 || line[0] == '#' {
				continue
			}

			if len(line) > 255 {
				return nil, fmt.Errorf("Too long line %q in %q found", line, file)
			}

			up := strings.Split(line, ":")
			if len(up) != 2 {
				return nil, fmt.Errorf("Invalid line %q in %q found", line, file)
			}

			if _, ok := ba.htFile[up[0]]; ok {
				return nil, fmt.Errorf("Multiple user %q in %q found", up[0], file)
			}

			switch pwdType := getPwdType(up[1]); pwdType {
			case pwdTypeApr1:
				fallthrough
			case pwdTypeMD5:
				prefix := pwdPrefixApr1
				if pwdType == pwdTypeMD5 {
					prefix = pwdPrefixMD5
				}

				parts := strings.SplitN(strings.TrimPrefix(up[1], prefix), "$", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("Malformed %q password %q in %q found", prefix, up[1], file)
				}

				ba.htFile[up[0]] = pwd{
					pwdHashed: []byte(parts[1]),
					pwdOrig:   []byte(up[1]),
					pwdPrefix: prefix,
					pwdSalt:   parts[0],
					pwdType:   pwdType,
				}
			case pwdTypeBcrypt:
				ba.htFile[up[0]] = pwd{
					pwdHashed: []byte(up[1]),
					pwdType:   pwdType,
				}
			default:
				return nil, fmt.Errorf("Unsupported password algorithm in %q found", file)
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	return ba, nil
}

// Validate implements the AccessControl interface
func (ba *BasicAuth) Validate(req *http.Request) error {
	auth := req.Header.Get(authHeader)
	if auth == "" {
		return fmt.Errorf("Empty %q HTTP header field given", authHeader)
	}

	if l := len("Basic "); l < len(auth) {
		auth = strings.TrimSpace(auth[l:])
	} else {
		return fmt.Errorf("Invalid %q HTTP header field given", authHeader)
	}

	decoded, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return err
	}

	up := strings.Split(string(decoded), ":")
	if len(up) != 2 {
		return fmt.Errorf("Invalid %q HTTP header field given", authHeader)
	}

	if ba.user == up[0] {
		if subtle.ConstantTimeCompare([]byte(ba.user), []byte(up[1])) == 1 {
			return nil
		}

		return fmt.Errorf("Invalid access data given")
	}

	if validateAccessData(up[0], up[1], ba.htFile) {
		return nil
	}

	return fmt.Errorf("Invalid access data given")
}
