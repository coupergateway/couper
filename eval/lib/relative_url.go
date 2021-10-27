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
	RelativeUrlFunc = newRelativeUrlFunction()
)

func newRelativeUrlFunction() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{{
			Name: "s",
			Type: cty.String,
		}},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			absURL := strings.TrimSpace(args[0].AsString())

			if !strings.HasPrefix(absURL, "/") && !strings.HasPrefix(absURL, "http://") && !strings.HasPrefix(absURL, "https://") {
				return cty.StringVal(""), fmt.Errorf("invalid url given: '%s'", absURL)
			}

			// Do not use the result of url.ParseRequestURI() above,
			// to preserve the URL encoding, e.g. in the query.
			if _, err := url.Parse(absURL); err != nil {
				return cty.StringVal(""), err
			}

			// The regexParseURL garanties the len of 10 in the result.
			urlParts := regexParseURL.FindStringSubmatch(absURL)

			// The path must begin w/ a slash.
			if !strings.HasPrefix(urlParts[5], "/") {
				urlParts[5] = "/" + urlParts[5]
			}

			return cty.StringVal(urlParts[5] + urlParts[6] + urlParts[8]), nil
		},
	})
}
