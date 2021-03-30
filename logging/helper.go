package logging

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

func filterHeader(list []string, src http.Header) map[string]string {
	header := make(map[string]string)
	for _, key := range list {
		ck := http.CanonicalHeaderKey(key)
		val, ok := src[ck]
		if !ok || len(val) == 0 || val[0] == "" {
			continue
		}
		header[strings.ToLower(key)] = strings.Join(val, "|")
	}
	return header
}

func splitHostPort(hp string) (string, string) {
	host, port, err := net.SplitHostPort(hp)
	if err != nil {
		return hp, port
	}
	return host, port
}

func roundMS(d time.Duration) string {
	const maxDuration time.Duration = 1<<63 - 1
	if d == maxDuration {
		return "0.0"
	}
	return fmt.Sprintf("%.3f", math.Round(float64(d)*1000)/1000/float64(time.Millisecond))
}
