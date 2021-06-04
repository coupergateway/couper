package docs_test

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestDocs_Links(t *testing.T) {
	helper := test.New(t)

	files, err := os.ReadDir("./")
	helper.Must(err)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}
		raw, readErr := ioutil.ReadFile(file.Name())
		helper.Must(readErr)

		regexLinks := regexp.MustCompile(`]\(#([^)]+)\)`)
		links := regexLinks.FindAllStringSubmatch(string(raw), -1)

		regexAnchors := regexp.MustCompile(`(?m)^#+ (.+)$`)
		anchors := regexAnchors.FindAllStringSubmatch(string(raw), -1)

		prepareAnchor := func(anchor string) string {
			anchor = strings.TrimSpace(anchor)
			anchor = strings.ToLower(anchor)
			anchor = strings.ReplaceAll(anchor, "`", "")
			anchor = strings.ReplaceAll(anchor, ".", "")
			anchor = strings.ReplaceAll(anchor, "(", "")
			anchor = strings.ReplaceAll(anchor, ")", "")
			anchor = strings.ReplaceAll(anchor, " ", "-")

			return anchor
		}

		// Search for ghost-anchors
		for _, link := range links {
			exists := false

			for _, anchor := range anchors {
				if link[1] == prepareAnchor(anchor[1]) {
					exists = true
					break
				}
			}

			if !exists {
				t.Errorf("%s: anchor for '%s' not found", file.Name(), link[1])
			}
		}

		// Search for ghost-links
		for _, anchor := range anchors {
			exists := false

			for _, link := range links {
				if prepareAnchor(anchor[1]) == link[1] {
					exists = true
					break
				}
			}

			if !exists && prepareAnchor(anchor[1]) != "table-of-contents" {
				t.Errorf("%s: link for '%s' not found", file.Name(), anchor[1])
			}
		}
	}
}
