package accesscontrol

import (
	"fmt"
	"math/bits"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

const (
	bearerType tokenSourceType = iota
	cookieType
	dpopType
	headerType
	valueType
)

type (
	tokenSourceType uint8
	// TokenSource represents the source from which a token is retrieved.
	TokenSource struct {
		expr   hcl.Expression
		name   string
		tsType tokenSourceType
	}
)

// NewTokenSource creates a new token source according to various configuration attributes.
func NewTokenSource(bearer, dpop bool, cookie, header string, value hcl.Expression) (*TokenSource, error) {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	var b uint8
	t := bearerType // default

	if bearer {
		b |= (1 << bearerType)
	}
	if dpop {
		b |= (1 << dpopType)
		t = dpopType
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

// TokenValue retrieves the token value from the request.
func (s *TokenSource) TokenValue(req *http.Request) (string, error) {
	var tokenValue string
	var err error

	switch s.tsType {
	case bearerType:
		tokenValue, err = getTokenFromAuthorization(req.Header, "Bearer")
	case dpopType:
		tokenValue, err = getTokenFromAuthorization(req.Header, "DPoP")
	case cookieType:
		cookie, cerr := req.Cookie(s.name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case headerType:
		if strings.ToLower(s.name) == "authorization" {
			tokenValue, err = getTokenFromAuthorization(req.Header, "Bearer")
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

// getTokenFromAuthorization retrieves a token for the given auth scheme from the Authorization request header field.
func getTokenFromAuthorization(reqHeaders http.Header, authScheme string) (string, error) {
	authorization := reqHeaders.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	pfx := strings.ToLower(authScheme) + " "
	if strings.HasPrefix(strings.ToLower(authorization), pfx) {
		return strings.Trim(authorization[len(pfx):], " "), nil
	}

	return "", fmt.Errorf("auth scheme %q required in authorization header", authScheme)
}
