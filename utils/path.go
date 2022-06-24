package utils

import (
	"path"
	"strings"
)

// JoinPath ensures the file-muxer behaviour for redirecting
// '/path' to '/path/' if not explicitly specified.
func JoinPath(elements ...string) string {
	if len(elements) == 0 {
		return ""
	}
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

func JoinOpenAPIPath(elements ...string) string {
	if len(elements) == 0 {
		return ""
	}

	if elements[len(elements)-1] == "/" {
		elements = elements[:len(elements)-1]
	}

	result := ""
	for _, element := range elements {
		if element == "" {
			element = "/"
		}
		resultHasSlashSuffix := result != "" && strings.HasSuffix(result, "/")
		if strings.HasPrefix(element, "/") && !resultHasSlashSuffix {
			result += "/"
		}

		result += strings.TrimPrefix(element, "/")
	}
	return result
}
