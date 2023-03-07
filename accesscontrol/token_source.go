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
	Invalid tokenSourceType = iota
	bearerType
	cookieType
	headerType
	valueType
)

type (
	tokenSourceType uint8
	tokenSource     struct {
		Expr hcl.Expression
		Name string
		Type tokenSourceType
	}
)

func newTokenSource(bearer bool, cookie, header string, value hcl.Expression) tokenSource {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			if bearer || h != "" || c != "" {
				return tokenSource{}
			}

			return tokenSource{
				Name: "",
				Type: valueType,
				Expr: value,
			}
		}
	}
	if c != "" && !bearer && h == "" {
		return tokenSource{
			Name: c,
			Type: cookieType,
		}
	}
	if h != "" && !bearer && c == "" {
		return tokenSource{
			Name: h,
			Type: headerType,
		}
	}
	if h == "" && c == "" {
		return tokenSource{
			Type: bearerType,
		}
	}
	return tokenSource{}
}

func (s tokenSource) TokenValue(req *http.Request) (string, error) {
	var tokenValue string
	var err error

	switch s.Type {
	case bearerType:
		if tokenValue = req.Header.Get("Authorization"); tokenValue != "" {
			if tokenValue, err = getBearer(tokenValue); err != nil {
				return "", errors.JwtTokenMissing.With(err)
			}
		}
	case cookieType:
		cookie, cerr := req.Cookie(s.Name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case headerType:
		if strings.ToLower(s.Name) == "authorization" {
			if tokenValue = req.Header.Get(s.Name); tokenValue != "" {
				if tokenValue, err = getBearer(tokenValue); err != nil {
					return "", errors.JwtTokenMissing.With(err)
				}
			}
		} else {
			tokenValue = req.Header.Get(s.Name)
		}
	case valueType:
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
