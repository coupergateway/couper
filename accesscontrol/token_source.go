package accesscontrol

import (
	"fmt"
	"math/bits"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

const (
	bearerType tokenSourceType = iota
	cookieType
	headerType
	valueType
)

type (
	tokenSourceType uint8
	TokenSource     struct {
		expr   hcl.Expression
		name   string
		tsType tokenSourceType
	}
)

func NewTokenSource(bearer bool, cookie, header string, value hcl.Expression) (*TokenSource, error) {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	var b uint8
	t := bearerType // default

	if bearer {
		b |= (1 << bearerType)
	}
	if c != "" {
		b |= (1 << cookieType)
		t = cookieType
	}
	if h != "" {
		b |= (1 << headerType)
		t = headerType
	}
	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			b |= (1 << valueType)
			t = valueType
		}
	}
	if bits.OnesCount8(b) > 1 {
		return nil, fmt.Errorf("only one of bearer, cookie, header or token_value attributes is allowed")
	}

	ts := &TokenSource{
		tsType: t,
	}
	switch t {
	case cookieType:
		ts.name = c
	case headerType:
		ts.name = h
	case valueType:
		ts.expr = value
	}

	return ts, nil
}

func (s *TokenSource) TokenValue(req *http.Request) (string, error) {
	var tokenValue string
	var err error

	switch s.tsType {
	case bearerType:
		if tokenValue = req.Header.Get("Authorization"); tokenValue != "" {
			if tokenValue, err = getBearer(tokenValue); err != nil {
				return "", errors.JwtTokenMissing.With(err)
			}
		}
	case cookieType:
		cookie, cerr := req.Cookie(s.name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case headerType:
		if strings.ToLower(s.name) == "authorization" {
			if tokenValue = req.Header.Get(s.name); tokenValue != "" {
				if tokenValue, err = getBearer(tokenValue); err != nil {
					return "", errors.JwtTokenMissing.With(err)
				}
			}
		} else {
			tokenValue = req.Header.Get(s.name)
		}
	case valueType:
		requestContext := eval.ContextFromRequest(req).HCLContext()
		value, err := eval.Value(requestContext, s.expr)
		if err != nil {
			return "", err
		}

		tokenValue = seetie.ValueToString(value)
	}

	if tokenValue == "" {
		return "", errors.JwtTokenMissing.Message("token required")
	}

	return tokenValue, nil
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", fmt.Errorf("bearer required with authorization header")
}
