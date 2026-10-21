package utils

import (
	"errors"
	"net/url"
	"strings"
)

// ErrPathAboveRoot reports a path whose dot segments resolve above the root.
var ErrPathAboveRoot = errors.New("path resolves above the root")

// NormalizeURLPath rewrites Path and RawPath of u to the normalized form of its
// path, so that EscapedPath keeps the encoding of the normalized form.
// It starts from RawPath, the path as sent. EscapedPath would re-encode Path when
// RawPath holds a byte like "|" that Go escapes, and that turns %2F into a slash.
func NormalizeURLPath(u *url.URL) error {
	escaped := u.RawPath
	if escaped == "" {
		escaped = u.EscapedPath()
	}
	normalized, err := NormalizePath(escaped)
	if err != nil {
		return err
	}
	if normalized == escaped {
		return nil
	}

	unescaped, err := url.PathUnescape(normalized)
	if err != nil {
		return err
	}
	u.RawPath = normalized
	u.Path = unescaped

	return nil
}

// NormalizePath removes dot segments and merges slashes of an escaped path
// (RFC 3986 section 5.2.4). Only a segment made of literal or %2E-encoded dots is
// decoded, every other byte is kept, so an encoded slash stays inside its segment.
// A ".." above the root is an error, not clamped.
func NormalizePath(escaped string) (string, error) {
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
				return "", ErrPathAboveRoot
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

// HasEmptyOrDotSegment reports whether a decoded path contains an empty or dot
// segment. A trailing slash is not an empty segment.
func HasEmptyOrDotSegment(path string) bool {
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
