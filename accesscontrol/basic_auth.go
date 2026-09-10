package accesscontrol

import (
	"bufio"
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
)

var _ AccessControl = &BasicAuth{}

// BasicAuth represents an AC-BasicAuth object
type BasicAuth struct {
	htFile       htData
	name         string
	user         string
	pass         string
	warnings     []Argon2CostWarning
	argon2       *Argon2Limiter
	argon2Memory uint32
}

// NewBasicAuth creates a new AC-BasicAuth object
func NewBasicAuth(name, user, pass, file string) (*BasicAuth, error) {
	ba := &BasicAuth{
		htFile: make(htData),
		name:   name,
		user:   user,
		pass:   pass,
		argon2: NewArgon2Limiter(DefaultArgon2MemoryBudget, 0),
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
			return nil, fmt.Errorf("parse error: invalid line: %d", lineNr)
		}

		username, password := up[0], up[1]

		if _, ok := ba.htFile[username]; ok {
			return nil, fmt.Errorf("multiple user: %s", username)
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
				return nil, fmt.Errorf("parse error: malformed password for user: %s", username)
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
		case pwdTypeArgon2id, pwdTypeArgon2i:
			prefix := pwdPrefixArgon2id
			if pwdType == pwdTypeArgon2i {
				prefix = pwdPrefixArgon2i
			}
			p, warnings, pErr := parseArgon2(password, prefix)
			if pErr != nil {
				return nil, fmt.Errorf("parse error: malformed password for user: %s: %w", username, pErr)
			}
			for _, w := range warnings {
				w.User, w.Line = username, lineNr
				ba.warnings = append(ba.warnings, w)
			}
			ba.htFile[username] = p
			ba.argon2Memory = max(ba.argon2Memory, p.argon2Memory)
		default:
			return nil, fmt.Errorf("parse error: algorithm not supported")
		}
	}

	if ba.argon2Memory > 0 {
		ba.argon2 = NewArgon2Limiter(DefaultArgon2MemoryBudget, ba.argon2Memory)
	}

	err = scanner.Err()
	return ba, err
}

// Argon2Memory returns the memory in KiB that one derivation of the most
// expensive argon2 entry needs, or 0 if no entry uses argon2.
func (ba *BasicAuth) Argon2Memory() uint32 {
	return ba.argon2Memory
}

// UseArgon2Limiter replaces the limiter, so all basic_auth blocks of a
// configuration share one memory budget.
func (ba *BasicAuth) UseArgon2Limiter(limiter *Argon2Limiter) {
	ba.argon2 = limiter
}

// Warnings lists the htpasswd entries that load with an argon2 parameter above
// the recommended maximum.
func (ba *BasicAuth) Warnings() []Argon2CostWarning {
	return ba.warnings
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

	if subtle.ConstantTimeCompare([]byte(ba.user), []byte(user)) == 1 {
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
		valid, vErr := validateAccessData(req.Context(), user, pass, ba.htFile, ba.argon2)
		if vErr != nil {
			return errors.BasicAuth.With(vErr).Message("file: argon2 verification abandoned")
		}
		if valid {
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
