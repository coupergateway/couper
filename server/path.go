package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coupergateway/couper/errors"
)

var errPathAboveRoot = fmt.Errorf("path resolves above the root")

// normalizeRequestPath rewrites req.URL to the canonical form of its path so that
// routing, access control and the upstream request agree on one resource.
func normalizeRequestPath(req *http.Request) *errors.Error {
	escaped := req.URL.EscapedPath()
	normalized, err := normalizePath(escaped)
	if err != nil {
		return errors.ClientRequest.With(err)
	}

	if normalized != escaped {
		unescaped, err := url.PathUnescape(normalized)
		if err != nil {
			return errors.ClientRequest.With(err)
		}
		req.URL.RawPath = normalized
		req.URL.Path = unescaped
	}

	// Only an encoded slash can leave an empty or dot segment in the decoded path.
	if hasEmptyOrDotSegment(req.URL.Path) {
		return errors.ClientRequest.Message("encoded slash forms an empty or dot segment")
	}

	return nil
}

// normalizePath removes dot segments and merges slashes of an escaped path
// (RFC 3986 section 5.2.4). Only a segment made of literal or %2E-encoded dots is
// decoded, every other byte is kept, so an encoded slash stays inside its segment.
// A ".." above the root is an error, not clamped.
func normalizePath(escaped string) (string, error) {
	if !strings.HasPrefix(escaped, "/") {
		return escaped, nil
	}

	segments := strings.Split(escaped[1:], "/")
	stack := make([]string, 0, len(segments))
	for _, segment := range segments {
		switch dotSegment(segment) {
		case 1:
			continue
		case 2:
			if len(stack) == 0 {
				return "", errPathAboveRoot
			}
			stack = stack[:len(stack)-1]
		default:
			if segment != "" {
				stack = append(stack, segment)
			}
		}
	}

	result := "/" + strings.Join(stack, "/")
	last := segments[len(segments)-1]
	if len(stack) > 0 && (last == "" || dotSegment(last) > 0) {
		result += "/"
	}

	return result, nil
}

// dotSegment reports the number of dots when the segment is a single or double
// dot segment in literal or %2E-encoded form, and 0 otherwise.
func dotSegment(segment string) int {
	dots := 0
	for i := 0; i < len(segment); {
		switch {
		case segment[i] == '.':
			i++
		case i+3 <= len(segment) && segment[i] == '%' && segment[i+1] == '2' && (segment[i+2] == 'e' || segment[i+2] == 'E'):
			i += 3
		default:
			return 0
		}
		dots++
		if dots > 2 {
			return 0
		}
	}

	return dots
}

// hasEmptyOrDotSegment reports whether a decoded path contains an empty or dot
// segment. A trailing slash is not an empty segment.
func hasEmptyOrDotSegment(path string) bool {
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, segment := range segments {
		switch segment {
		case ".", "..":
			return true
		case "":
			if i < len(segments)-1 {
				return true
			}
		}
	}

	return false
}
