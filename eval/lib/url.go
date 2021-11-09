package lib

import (
	"net/url"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	url1 "github.com/avenga/couper/eval/lib/url"
)

var (
	UrlEncodeFunc   = newUrlEncodeFunction()
	RelativeUrlFunc = newRelativeUrlFunction()
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

func newRelativeUrlFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "s",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			absURL := strings.TrimSpace(args[0].AsString())
			relURL, err := url1.RelativeURL(absURL)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(relURL), nil
		},
	})
}
