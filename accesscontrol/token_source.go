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
	BearerType TokenSourceType = iota
	CookieType
	HeaderType
	ValueType
)

type (
	TokenSourceType uint8
	tokenSource     struct {
		Expr hcl.Expression
		Name string
		Type TokenSourceType
	}
)

func NewTokenSource(bearer bool, cookie, header string, value hcl.Expression) (*tokenSource, error) {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	var b uint8
	t := BearerType // default

	if bearer {
		b |= (1 << BearerType)
	}
	if c != "" {
		b |= (1 << CookieType)
		t = CookieType
	}
	if h != "" {
		b |= (1 << HeaderType)
		t = HeaderType
	}
	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			b |= (1 << ValueType)
			t = ValueType
		}
	}
	if bits.OnesCount8(b) > 1 {
		return nil, fmt.Errorf("only one of bearer, cookie, header or token_value attributes is allowed")
	}

	ts := &tokenSource{
		Type: t,
	}
	switch t {
	case CookieType:
		ts.Name = c
	case HeaderType:
		ts.Name = h
	case ValueType:
		ts.Expr = value
	}

	return ts, nil
}

func (s tokenSource) TokenValue(req *http.Request) (string, error) {
	var tokenValue string
	var err error

	switch s.Type {
	case BearerType:
		if tokenValue = req.Header.Get("Authorization"); tokenValue != "" {
			if tokenValue, err = getBearer(tokenValue); err != nil {
				return "", errors.JwtTokenMissing.With(err)
			}
		}
	case CookieType:
		cookie, cerr := req.Cookie(s.Name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case HeaderType:
		if strings.ToLower(s.Name) == "authorization" {
			if tokenValue = req.Header.Get(s.Name); tokenValue != "" {
				if tokenValue, err = getBearer(tokenValue); err != nil {
					return "", errors.JwtTokenMissing.With(err)
				}
			}
		} else {
			tokenValue = req.Header.Get(s.Name)
		}
	case ValueType:
		requestContext := eval.ContextFromRequest(req).HCLContext()
		value, err := eval.Value(requestContext, s.Expr)
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
