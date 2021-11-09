package url

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	// https://datatracker.ietf.org/doc/html/rfc3986#page-50
	regexParseURL = regexp.MustCompile(`^(([^:/?#]+):)?(//([^/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?`)
)

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
