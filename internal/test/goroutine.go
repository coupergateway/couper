package test

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"runtime/pprof"
	"strconv"
)

// NumGoroutines counts all goroutines whose stack contains the filter. The
// profile groups goroutines by identical stack, so goroutines in the same
// function can appear in more than one group; all matching groups are summed.
func NumGoroutines(filter string) (numRoutine int) {
	profile := pprof.Lookup("goroutine")
	profileBuf := &bytes.Buffer{}
	_ = profile.WriteTo(profileBuf, 1)
	pr := bufio.NewReader(profileBuf)

	stackRegex := regexp.MustCompile(`(\d+)\s@\s0x`)
	groupCount := 0
	groupMatched := false
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
			groupCount, _ = strconv.Atoi(string(match[1]))
			groupMatched = false
			continue
		}
		if !groupMatched && bytes.Contains(line, []byte(filter)) {
			numRoutine += groupCount
			groupMatched = true
		}
	}
	return numRoutine
}
