package writer

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	SecureCookiesStrip = "strip"
	SecureCookieAV     = "Secure"
	setCookieHeader    = "Set-Cookie"
)

var regexSplitSetCookie = regexp.MustCompile(`([^;]+);?`)

func stripSecureCookies(header http.Header) {
	list := header.Values(setCookieHeader)
	header.Del(setCookieHeader)

	for _, original := range list {
		parts, isSecure := parseSetCookieHeader(original)

		if !isSecure {
			header.Add(setCookieHeader, original) // Unchanged
		} else {
			header.Add(setCookieHeader, strings.Join(parts, "; "))
		}
	}
}

// parseSetCookieHeader splits the given Set-Cookie HTTP header field value
// and always removes the <Secure> flag. If the <Secure> flag was present, the
// second return value is set to <true>, otherwise to <false>.
func parseSetCookieHeader(setCookie string) ([]string, bool) {
	var parts []string
	var isSecure bool

	for _, m := range regexSplitSetCookie.FindAllStringSubmatch(setCookie, -1) {
		part := strings.TrimSpace(m[1])

		if strings.EqualFold(part, SecureCookieAV) {
			isSecure = true

			continue
		}

		parts = append(parts, part)
	}

	return parts, isSecure
}
