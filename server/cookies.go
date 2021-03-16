package server

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	SecureCookiesStrip   = "strip"
	SecureCookiesEnforce = "enforce"
	SecureCookieAV       = "Secure"
	setCookieHeader      = "Set-Cookie"
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

func enforceSecureCookies(header http.Header) {
	list := header.Values(setCookieHeader)
	header.Del(setCookieHeader)

	for _, original := range list {
		parts, isSecure := parseSetCookieHeader(original)

		if isSecure {
			header.Add(setCookieHeader, original) // Unchanged
		} else {
			header.Add(
				setCookieHeader, strings.Join(append(parts, SecureCookieAV), "; "),
			)
		}
	}
}

func parseSetCookieHeader(setCookie string) ([]string, bool) {
	var parts []string
	var isSecure bool

	for _, m := range regexSplitSetCookie.FindAllStringSubmatch(setCookie, -1) {
		part := strings.TrimSpace(m[1])

		if strings.ToLower(part) == strings.ToLower(SecureCookieAV) {
			isSecure = true

			continue
		}

		parts = append(parts, part)
	}

	return parts, isSecure
}
