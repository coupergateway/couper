package accesscontrol

import (
	"fmt"
	"math/bits"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

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
		tokenValue, err = getBearerAuth(req.Header)
	case cookieType:
		cookie, cerr := req.Cookie(s.name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case headerType:
		if strings.ToLower(s.name) == "authorization" {
			tokenValue, err = getBearerAuth(req.Header)
		} else {
			tokenValue = req.Header.Get(s.name)
		}
	case valueType:
		requestContext := eval.ContextFromRequest(req).HCLContext()
		var value cty.Value
		value, err = eval.Value(requestContext, s.expr)
		if err != nil {
			return "", err
		}

		tokenValue = seetie.ValueToString(value)
	}

	if err != nil {
		return "", err
	}

	if tokenValue == "" {
		return "", fmt.Errorf("token required")
	}

	return tokenValue, nil
}

func getBearerAuth(reqHeaders http.Header) (string, error) {
	authorization := reqHeaders.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	tokenValue, err := getBearer(authorization)
	if err != nil {
		return "", err
	}
	return tokenValue, nil
}

func getBearer(authorization string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(authorization), bearer) {
		return strings.Trim(authorization[len(bearer):], " "), nil
	}
	return "", fmt.Errorf("bearer with token required in authorization header")
}
