package lib

import (
	"net/url"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	UrlEncodeFunc = newUrlEncodeFunction()
)

func newUrlEncodeFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "s",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			first := args[0]
			result := strings.Replace(url.QueryEscape(first.AsString()), "+", "%20", -1)
			return cty.StringVal(string(result)), nil
		},
	})
}

func AbsoluteURL(urlRef string, origin *url.URL) (string, error) {
	u, err := url.Parse(urlRef)
	if err != nil {
		return "", err
	}

	if !u.IsAbs() {
		return origin.ResolveReference(u).String(), nil
	}
	return urlRef, nil
}
