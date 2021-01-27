package docs_test

import (
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestDocs_Links(t *testing.T) {
	helper := test.New(t)

	raw, err := ioutil.ReadFile("README.md")
	helper.Must(err)

	regexLinks := regexp.MustCompile(`]\(#([^)]+)\)`)
	links := regexLinks.FindAllStringSubmatch(string(raw), -1)

	regexAnchors := regexp.MustCompile(`<a name="([^"]+)"></a>`)
	anchors := regexAnchors.FindAllStringSubmatch(string(raw), -1)

	// Search for ghost-anchors
	for _, link := range links {
		exists := false

		for _, anchor := range anchors {
			if link[1] == anchor[1] {
				exists = true
				break
			}
		}

		if !exists {
			t.Errorf("Anchor for '%s' not found", link[1])
		}
	}

	// Search for ghost-links
	for _, anchor := range anchors {
		exists := false

		for _, link := range links {
			if anchor[1] == link[1] {
				exists = true
				break
			}
		}

		if !exists {
			t.Errorf("Link for '%s' not found", anchor[1])
		}
	}
}
