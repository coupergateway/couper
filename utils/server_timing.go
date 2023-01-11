package utils

import (
	"fmt"
	"strings"
)

type ServerTimings map[string]string

// CollectMetricNames collects metric names from the given Server-Timing HTTP header field.
// See https://www.w3.org/TR/server-timing/
func CollectMetricNames(header string, timings ServerTimings) {
	metrics := splitToMetrics(header)

	for _, metric := range metrics {
		metricName := ""

		for len(metric) > 0 && isTokenChar(metric[0]) {
			metricName += string(metric[0])

			metric = metric[1:]
		}

		if metricName == "" {
			continue
		}

		timings[metricName] = ""
	}
}

// MergeMetrics merges timings from 'src' into 'dest'.
func MergeMetrics(src, dest ServerTimings) {
	for k, v := range src {
		key := k

		if _, exists := dest[key]; exists {
			key = uniqueKey(key, dest)
		}

		dest[key] = v
	}
}

func splitToMetrics(header string) []string {
	var (
		part  string
		parts []string
	)

	// Trim WS and ','
	trimLeft := func(s string) string {
		return strings.TrimLeft(s, string([]byte{0, 9, 10, 11, 13, 32, 44}))
	}

	header = trimLeft(header)

	for len(header) > 0 {
		if header[0] == '"' {
			// Consume '"' character
			part += string(header[0])

			header = header[1:]

			for len(header) > 0 && header[0] != '"' {
				part += string(header[0])

				header = header[1:]
			}

			if len(header) == 0 {
				parts = append(parts, strings.TrimSpace(part))

				break
			}

			// Consume '"' character
			part += string(header[0])

			header = header[1:]
		} else if header[0] == ',' {
			parts = append(parts, strings.TrimSpace(part))

			// Trim WS and ','
			header = strings.TrimLeft(header, string([]byte{0, 9, 10, 11, 13, 32, 44}))

			// Reset
			part = ""
		} else {
			part += string(header[0])

			header = header[1:]
		}
	}

	if part := strings.TrimSpace(part); part != "" {
		parts = append(parts, part)
	}

	return parts
}

func uniqueKey(key string, timings ServerTimings) string {
	i := 1

	for {
		newKey := fmt.Sprintf("%s_%d", key, i)

		if _, exists := timings[newKey]; !exists {
			return newKey
		}

		i++
	}
}

// RFC 2616
func isTokenChar(ch byte) bool {
	const separators = `()<>@,;:\"/[]?={}` + " " + "\t"

	if ch <= 31 || ch == 127 {
		return false
	}

	return !strings.Contains(separators, string(ch))
}
