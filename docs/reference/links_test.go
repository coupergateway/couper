package reference_test

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

type list map[string]interface{}

type item struct {
	links   list
	anchors list
}

var (
	regexLinks   = regexp.MustCompile(`(\[[^]]+\])\(([^#)]+\.md)?(#[^)]+)?\)`)
	regexAnchors = regexp.MustCompile(`(?m)^#+ (.+)$`)
)

func Test_Links(t *testing.T) {
	helper := test.New(t)

	dir, err := filepath.Abs("./")
	helper.Must(err)

	data := make(list)
	parseFiles(t, string(http.Dir(dir)), data, helper)

	for file, i := range data {
		set := i.(*item)

		for link, a := range set.links {
			anchors := a.([]string)

			if _, ok := data[link]; !ok {
				t.Errorf("Link to '%s' referenced in '%s' does not exists.", link, file)
			}

			for _, anchor := range anchors {
				if anchor != "" {
					if _, ok := data[link].(*item).anchors[anchor]; !ok {
						t.Errorf("Link to '%s%s' referenced in '%s' does not exists.", link, anchor, file)
					}
				}
			}
		}
	}
}

func parseFiles(t *testing.T, dir string, data list, helper *test.Helper) {
	files, err := os.ReadDir(dir)
	helper.Must(err)

	for _, file := range files {
		abs := dir + "/" + file.Name()

		if file.IsDir() {
			parseFiles(t, abs, data, helper)
			continue
		}

		if !isMarkdownFilename(file.Name()) {
			continue
		}

		raw, err := os.ReadFile(abs)
		helper.Must(err)

		matches := regexLinks.FindAllStringSubmatch(string(raw), -1)

		links := make(list)
		for _, set := range matches {
			if len(set) != 4 || set[2] == "" && set[3] == "" {
				t.Fatalf("Unexpected match: %#v", set)
			}

			link := abs
			if set[2] != "" {
				if strings.HasPrefix(set[2], "http://") || strings.HasPrefix(set[2], "https://") {
					continue
				}

				link = path.Join(path.Dir(link), set[2])
			}

			if _, ok := links[link]; ok {
				links[link] = append(links[link].([]string), set[3])
			} else {
				links[link] = []string{set[3]}
			}
		}

		matches = regexAnchors.FindAllStringSubmatch(string(raw), -1)

		anchors := make(list)
		for _, match := range matches {
			anchors[prepareAnchor(t, match[1])] = true
		}

		data[abs] = &item{links: links, anchors: anchors}

		// Debug: fmt.Printf("%s\n>>> %#v\n>>>%#v\n\n", abs, links, anchors)
	}
}

func prepareAnchor(t *testing.T, anchor string) string {
	if matches := regexLinks.FindAllStringSubmatch(anchor, -1); len(matches) == 1 {
		if !strings.HasPrefix(matches[0][1], "[") || !strings.HasSuffix(matches[0][1], "]") {
			t.Fatalf("Unexpected match: %#v", matches[0])
		}

		anchor = matches[0][1][1 : len(matches[0][1])-1]
	}

	anchor = strings.TrimSpace(anchor)
	anchor = strings.ToLower(anchor)
	anchor = strings.ReplaceAll(anchor, "`", "")
	anchor = strings.ReplaceAll(anchor, ".", "")
	anchor = strings.ReplaceAll(anchor, ":", "")
	anchor = strings.ReplaceAll(anchor, "(", "")
	anchor = strings.ReplaceAll(anchor, ")", "")
	anchor = strings.ReplaceAll(anchor, "~", "")
	anchor = strings.ReplaceAll(anchor, " ", "-")

	return "#" + anchor
}

func isMarkdownFilename(name string) bool {
	return strings.HasSuffix(name, ".md")
}
