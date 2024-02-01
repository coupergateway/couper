package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/coupergateway/couper/internal/test"
)

// TestChangelog_Links ensures that every added/changed/fixed listing has a link (pr/issue).
func TestChangelog_Links(t *testing.T) {
	helper := test.New(t)

	changelog, err := os.Open("CHANGELOG.md")
	helper.Must(err)
	defer changelog.Close()

	r := bufio.NewReader(changelog)

	linkRegex := regexp.MustCompile(`\(\[#\d+]\(.+/(pull|issues)/\d+\)\)\n$`)

	nr := 0
	for {
		nr++
		line, readErr := r.ReadSlice('\n')
		if readErr == io.EOF {
			return
		}
		helper.Must(readErr)

		// Link Condition introduced with 1.2 release.
		if bytes.Equal(line, []byte("Release date: 2021-04-21\n")) {
			return
		}

		if !bytes.HasPrefix(line, []byte("  * ")) {
			continue
		}

		if !linkRegex.Match(line) {
			t.Errorf("line %d: missing issue or pull-request link:\n\t%q", nr, string(line))
		}
	}
}
