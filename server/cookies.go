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

	for _, sc := range list {
		var parts []string
		var isSec bool

		for _, part := range parseSetCookieHeader(sc) {
			if strings.ToLower(part) == strings.ToLower(SecureCookieAV) {
				isSec = true
				continue
			}

			parts = append(parts, part)
		}

		if !isSec {
			header.Add(setCookieHeader, sc) // Unchanged
		} else {
			header.Add(setCookieHeader, strings.Join(parts, "; "))
		}
	}
}

func enforceSecureCookies(header http.Header) {
	list := header.Values(setCookieHeader)
	header.Del(setCookieHeader)

	for _, sc := range list {
		var parts []string
		var isSec bool

		for _, part := range parseSetCookieHeader(sc) {
			if strings.ToLower(part) == strings.ToLower(SecureCookieAV) {
				isSec = true
				continue
			}

			parts = append(parts, part)
		}

		if isSec {
			header.Add(setCookieHeader, sc) // Unchanged
		} else {
			header.Add(
				setCookieHeader, strings.Join(append(parts, SecureCookieAV), "; "),
			)
		}
	}
}

func parseSetCookieHeader(setCookie string) []string {
	var parts []string

	for _, m := range regexSplitSetCookie.FindAllStringSubmatch(setCookie, -1) {
		parts = append(parts, strings.TrimSpace(m[1]))
	}

	return parts
}
