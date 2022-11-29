package logging

import (
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

func RoundMS(d time.Duration) float64 {
	const (
		maxDuration time.Duration = 1<<63 - 1
		milliSecond               = float64(time.Millisecond)
	)

	if d == maxDuration {
		return 0.0
	}

	return math.Round((float64(d)/milliSecond)*1000) / 1000
}
