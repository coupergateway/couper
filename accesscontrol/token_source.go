package accesscontrol

import (
	"fmt"
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

	types := make(map[TokenSourceType]struct{})
	if bearer {
		types[BearerType] = struct{}{}
	}
	if c != "" {
		types[CookieType] = struct{}{}
	}
	if h != "" {
		types[HeaderType] = struct{}{}
	}
	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			types[ValueType] = struct{}{}
		}
	}
	if len(types) > 1 {
		return nil, fmt.Errorf("only one of bearer, cookie, header or token_value attributes is allowed")
	}

	if _, ok := types[ValueType]; ok {
		return &tokenSource{
			Expr: value,
			Type: ValueType,
		}, nil
	}
	if _, ok := types[CookieType]; ok {
		return &tokenSource{
			Name: c,
			Type: CookieType,
		}, nil
	}
	if _, ok := types[HeaderType]; ok {
		return &tokenSource{
			Name: h,
			Type: HeaderType,
		}, nil
	}
	// default
	return &tokenSource{
		Type: BearerType,
	}, nil
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
