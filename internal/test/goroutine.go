package test

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"runtime/pprof"
	"strconv"
)

func NumGoroutines(filter string) (numRoutine int) {
	profile := pprof.Lookup("goroutine")
	profileBuf := &bytes.Buffer{}
	_ = profile.WriteTo(profileBuf, 1)
	pr := bufio.NewReader(profileBuf)

	stackRegex := regexp.MustCompile(`(\d+)\s@\s0x`)
	for {
		line, _, readErr := pr.ReadLine()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			panic(readErr)
		}
		match := stackRegex.FindSubmatch(line)
		if len(match) > 1 {
			numRoutine, _ = strconv.Atoi(string(match[1]))
			continue
		}
		if bytes.Contains(line, []byte(filter)) {
			return numRoutine
		}
	}
	return -1
}
