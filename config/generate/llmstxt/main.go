//go:build exclude

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	contentDir = "docs/website/content"
	outputFile = "docs/website/static/llms.txt"
	baseURL    = "https://docs.couper.io"
)

type page struct {
	title       string
	description string
	urlPath     string
	slug        string
	weight      int
}

// section defines a group of pages for the llms.txt output.
type section struct {
	heading string
	pages   []page
}

func main() {
	sections := []section{
		{heading: "Getting Started", pages: collectPages(filepath.Join(contentDir, "getting-started"))},
		{heading: "Configuration", pages: collectPages(filepath.Join(contentDir, "configuration"))},
		{heading: "Configuration Blocks", pages: collectPages(filepath.Join(contentDir, "configuration", "block"))},
		{heading: "Observation", pages: collectPages(filepath.Join(contentDir, "observation"))},
	}

	var buf bytes.Buffer

	buf.WriteString("# Couper\n\n")
	buf.WriteString("> Couper is a lightweight open-source API gateway that acts as an entry point for clients ")
	buf.WriteString("and an exit point to upstream services. It adds access control, observability, and backend ")
	buf.WriteString("connectivity on a separate configuration layer using HCL 2.0 syntax.\n\n")
	buf.WriteString("Install via Docker (`docker pull coupergateway/couper`) or Homebrew (`brew install couper`). ")
	buf.WriteString("Supports file/SPA serving, API proxying, request/response manipulation, JWT, Basic Auth, ")
	buf.WriteString("OAuth2, OIDC, SAML, TLS/mTLS, WebSockets, CORS, rate limiting, and Prometheus metrics.\n")

	for _, sec := range sections {
		if len(sec.pages) == 0 {
			continue
		}
		buf.WriteString("\n## " + sec.heading + "\n\n")
		for _, p := range sec.pages {
			line := fmt.Sprintf("- [%s](%s%s)", p.title, baseURL, p.urlPath)
			if p.description != "" {
				line += ": " + p.description
			}
			buf.WriteString(line + "\n")
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(outputFile, buf.Bytes(), 0644); err != nil {
		panic(err)
	}

	fmt.Println("Generated " + outputFile)
}

func collectPages(dir string) []page {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read %s: %v\n", dir, err)
		return nil
	}

	var pages []page
	for _, e := range entries {
		if e.IsDir() || e.Name() == "_index.md" {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, e.Name())
		p := parsePage(filePath, dir)
		if p.title != "" {
			pages = append(pages, p)
		}
	}

	sort.Slice(pages, func(i, j int) bool {
		if pages[i].weight != pages[j].weight {
			if pages[i].weight == 0 {
				return false
			}
			if pages[j].weight == 0 {
				return true
			}
			return pages[i].weight < pages[j].weight
		}
		return pages[i].title < pages[j].title
	})

	return pages
}

func parsePage(filePath, dir string) page {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return page{}
	}

	title, description, slug, weight := parseFrontmatter(content)

	if title == "" {
		h1Re := regexp.MustCompile(`(?m)^#\s+(.+)$`)
		if m := h1Re.FindSubmatch(content); len(m) > 1 {
			title = string(m[1])
		}
	}

	if title == "" {
		return page{}
	}

	if description == "" {
		description = extractFirstParagraph(content)
	}

	// Build URL path from content directory structure.
	relDir := strings.TrimPrefix(dir, contentDir)
	fileName := strings.TrimSuffix(filepath.Base(filePath), ".md")
	if slug != "" {
		fileName = slug
	}
	urlPath := relDir + "/" + fileName

	return page{
		title:       strings.Trim(title, "'\""),
		description: strings.Trim(description, "'\""),
		urlPath:     urlPath,
		slug:        slug,
		weight:      weight,
	}
}

func parseFrontmatter(content []byte) (title, description, slug string, weight int) {
	sep := []byte("---")
	if !bytes.HasPrefix(content, sep) {
		return
	}

	endIdx := bytes.Index(content[3:], sep)
	if endIdx < 0 {
		return
	}

	s := bufio.NewScanner(bytes.NewReader(content[3 : 3+endIdx]))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "title:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		} else if strings.HasPrefix(line, "slug:") {
			slug = strings.TrimSpace(strings.TrimPrefix(line, "slug:"))
			slug = strings.Trim(slug, "'\"")
		} else if strings.HasPrefix(line, "weight:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "weight:"))
			fmt.Sscanf(val, "%d", &weight)
		}
	}
	return
}

// extractFirstParagraph gets the first text paragraph after the H1 heading,
// stopping at tables, shortcodes, or double blank lines.
func extractFirstParagraph(content []byte) string {
	lines := bytes.Split(content, []byte("\n"))
	var pastFrontmatter, pastH1 bool
	var frontmatterCount int
	var result strings.Builder
	var emptyLines int

	for _, line := range lines {
		lineStr := strings.TrimSpace(string(line))

		if lineStr == "---" {
			frontmatterCount++
			if frontmatterCount >= 2 {
				pastFrontmatter = true
			}
			continue
		}
		if !pastFrontmatter {
			continue
		}

		if strings.HasPrefix(lineStr, "# ") {
			pastH1 = true
			continue
		}
		if !pastH1 {
			continue
		}

		if strings.HasPrefix(lineStr, "|") ||
			strings.HasPrefix(lineStr, "{{<") ||
			strings.HasPrefix(lineStr, "##") ||
			strings.HasPrefix(lineStr, "<!--") {
			break
		}

		if lineStr == "" {
			emptyLines++
			if emptyLines >= 2 && result.Len() > 0 {
				break
			}
			continue
		}

		emptyLines = 0

		if strings.HasPrefix(lineStr, ">") {
			text := strings.TrimSpace(strings.TrimPrefix(lineStr, ">"))
			// Skip emoji-only blockquotes.
			if len(text) <= 5 {
				continue
			}
			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(text)
			continue
		}

		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(lineStr)
	}

	desc := result.String()

	// Strip markdown formatting.
	desc = strings.ReplaceAll(desc, "`", "")
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	desc = linkRe.ReplaceAllString(desc, "$1")

	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return desc
}
