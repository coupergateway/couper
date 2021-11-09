package lib

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	// https://datatracker.ietf.org/doc/html/rfc3986#page-50
	regexParseURL   = regexp.MustCompile(`^(([^:/?#]+):)?(//([^/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?`)
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
			relURL, err := RelativeURL(absURL)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(relURL), nil
		},
	})
}

func RelativeURL(absURL string) (string, error) {
	if !strings.HasPrefix(absURL, "/") && !strings.HasPrefix(absURL, "http://") && !strings.HasPrefix(absURL, "https://") {
		return "", fmt.Errorf("invalid url given: %q", absURL)
	}

	// Do not use the result of url.Parse() to preserve the # character in an emtpy fragment.
	if _, err := url.Parse(absURL); err != nil {
		return "", err
	}

	// The regexParseURL garanties the len of 10 in the result.
	urlParts := regexParseURL.FindStringSubmatch(absURL)

	// The path must begin w/ a slash.
	if !strings.HasPrefix(urlParts[5], "/") {
		urlParts[5] = "/" + urlParts[5]
	}

	return urlParts[5] + urlParts[6] + urlParts[8], nil
}
