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
	Invalid TokenSourceType = iota
	Cookie
	Header
	Value
)

type (
	TokenSourceType uint8
	TokenSource     struct {
		Expr hcl.Expression
		Name string
		Type TokenSourceType
	}
)

func NewTokenSource(cookie, header string, value hcl.Expression) TokenSource {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			if h != "" || c != "" {
				return TokenSource{}
			}

			return TokenSource{
				Name: "",
				Type: Value,
				Expr: value,
			}
		}
	}
	if c != "" && h == "" {
		return TokenSource{
			Name: c,
			Type: Cookie,
		}
	}
	if h != "" && c == "" {
		return TokenSource{
			Name: h,
			Type: Header,
		}
	}
	if h == "" && c == "" {
		return TokenSource{
			Name: "Authorization",
			Type: Header,
		}
	}
	return TokenSource{}
}

func (s TokenSource) TokenValue(req *http.Request) (string, error) {
	var tokenValue string
	var err error

	switch s.Type {
	case Cookie:
		cookie, cerr := req.Cookie(s.Name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case Header:
		if strings.ToLower(s.Name) == "authorization" {
			if tokenValue = req.Header.Get(s.Name); tokenValue != "" {
				if tokenValue, err = getBearer(tokenValue); err != nil {
					return "", errors.JwtTokenMissing.With(err)
				}
			}
		} else {
			tokenValue = req.Header.Get(s.Name)
		}
	case Value:
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
