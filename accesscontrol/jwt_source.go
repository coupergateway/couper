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
	Invalid JWTSourceType = iota
	Cookie
	Header
	Value
)

type (
	JWTSourceType uint8
	JWTSource     struct {
		Expr hcl.Expression
		Name string
		Type JWTSourceType
	}
)

func NewJWTSource(cookie, header string, value hcl.Expression) JWTSource {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			if h != "" || c != "" {
				return JWTSource{}
			}

			return JWTSource{
				Name: "",
				Type: Value,
				Expr: value,
			}
		}
	}
	if c != "" && h == "" {
		return JWTSource{
			Name: c,
			Type: Cookie,
		}
	}
	if h != "" && c == "" {
		return JWTSource{
			Name: h,
			Type: Header,
		}
	}
	if h == "" && c == "" {
		return JWTSource{
			Name: "Authorization",
			Type: Header,
		}
	}
	return JWTSource{}
}

func (s JWTSource) TokenValue(req *http.Request) (string, error) {
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
