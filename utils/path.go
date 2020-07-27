package utils

import (
	"path"
	"strings"
)

// JoinPath ensures the file-muxer behaviour for redirecting
// '/path' to '/path/' if not explicitly specified.
func JoinPath(elements ...string) string {
	suffix := "/"
	if !strings.HasSuffix(elements[len(elements)-1], "/") {
		suffix = ""
	}

	path := path.Join(elements...)
	if path == "/" {
		return path
	}

	return path + suffix
}
