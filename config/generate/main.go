//go:build exclude

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/meta"
)

type entry struct {
	Attributes  []interface{} `json:"attributes"`
	Blocks      []interface{} `json:"blocks"`
	Description string        `json:"description"`
	ID          string        `json:"objectID"`
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	URL         string        `json:"url"`
}

type attr struct {
	Default     string `json:"default"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Type        string `json:"type"`
}

type block struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

const (
	searchAppID     = "MSIN2HU7WH"
	searchIndex     = "docs"
	searchClientKey = "SEARCH_CLIENT_API_KEY"

	configurationPath = "docs/website/content/configuration"
	docsBlockPath     = configurationPath + "/block"

	urlBasePath = "/configuration/"
)

// export md: 1) search for ::attribute, replace if exist or append at end
func main() {

	client := search.NewClient(searchAppID, os.Getenv(searchClientKey))
	index := client.InitIndex(searchIndex)

	filenameRegex := regexp.MustCompile(`(URL|JWT|OpenAPI|[a-z0-9]+)`)
	bracesRegex := regexp.MustCompile(`{([^}]*)}`)

	attributesMap := map[string][]reflect.StructField{
		"RequestHeadersAttributes":  newFields(&meta.RequestHeadersAttributes{}),
		"ResponseHeadersAttributes": newFields(&meta.ResponseHeadersAttributes{}),
		"FormParamsAttributes":      newFields(&meta.FormParamsAttributes{}),
		"QueryParamsAttributes":     newFields(&meta.QueryParamsAttributes{}),
		"LogFieldsAttribute":        newFields(&meta.LogFieldsAttribute{}),
	}

	blockNamesMap := map[string]string{
		"oauth2_ac":       "beta_oauth2",
		"oauth2_req_auth": "oauth2",
	}

	processedFiles := make(map[string]struct{})
	var allEntries []entry // Collect all entries to index after clearing

	for _, impl := range []interface{}{
		&config.API{},
		&config.Backend{},
		&config.BackendTLS{},
		&config.BasicAuth{},
		&config.CORS{},
		&config.Defaults{},
		&config.Definitions{},
		&config.Endpoint{},
		&config.ErrorHandler{},
		&config.Files{},
		&config.Health{},
		&config.JWTSigningProfile{},
		&config.JWT{},
		&config.Job{},
		&config.OAuth2AC{},
		&config.OAuth2ReqAuth{},
		&config.OIDC{},
		&config.OpenAPI{},
		&config.Proxy{},
		&config.RateLimit{},
		&config.RateLimiter{},
		&config.Request{},
		&config.Response{},
		&config.SAML{},
		&config.Server{},
		&config.ClientCertificate{},
		&config.ServerCertificate{},
		&config.ServerTLS{},
		&config.Settings{},
		&config.Spa{},
		&config.TokenRequest{},
		&config.Websockets{},
	} {
		t := reflect.TypeOf(impl).Elem()
		name := reflect.TypeOf(impl).String()
		name = strings.TrimPrefix(name, "*config.")
		blockName := strings.ToLower(strings.Trim(filenameRegex.ReplaceAllString(name, "${1}_"), "_"))

		if _, exists := blockNamesMap[blockName]; exists {
			blockName = blockNamesMap[blockName]
		}

		urlPath, _ := url.JoinPath(urlBasePath, "block", blockName)

		// Extract description from the block's markdown file
		blockDescription := extractBlockDescription(blockName)

		result := entry{
			Name:        blockName,
			URL:         strings.ToLower(urlPath),
			Type:        "block",
			Description: blockDescription,
		}

		result.ID = result.URL

		var fields []reflect.StructField
		fields = collectFields(t, fields)

		inlineType, ok := impl.(config.Inline)
		if ok {
			it := reflect.TypeOf(inlineType.Inline()).Elem()
			for i := 0; i < it.NumField(); i++ {
				field := it.Field(i)
				if _, ok := attributesMap[field.Name]; ok {
					fields = append(fields, attributesMap[field.Name]...)
				} else {
					fields = append(fields, field)
				}
			}
		}

		for _, field := range fields {
			if field.Tag.Get("docs") == "" {
				continue
			}

			hclParts := strings.Split(field.Tag.Get("hcl"), ",")
			if len(hclParts) == 0 {
				continue
			}

			name := hclParts[0]
			fieldDescription := field.Tag.Get("docs")
			fieldDescription = bracesRegex.ReplaceAllString(fieldDescription, "`${1}`")

			if len(hclParts) > 1 && hclParts[1] == "block" {
				b := block{
					Description: fieldDescription,
					Name:        name,
				}
				result.Blocks = append(result.Blocks, b)
				continue
			}

			fieldType := field.Tag.Get("type")
			if fieldType == "" {
				ft := strings.Replace(field.Type.String(), "*", "", 1)
				if ft == "config.List" {
					ft = "[]string"
				}
				if ft[:2] == "[]" {
					ft = "tuple (" + ft[2:] + ")"
				} else if strings.Contains(ft, "int") {
					ft = "number"
				} else if ft != "string" && ft != "bool" {
					ft = "object"
				}
				fieldType = ft
			}

			fieldDefault := field.Tag.Get("default")
			if fieldDefault == "" && fieldType == "bool" {
				fieldDefault = "false"
			} else if fieldDefault == "" && strings.HasPrefix(fieldType, "tuple ") {
				fieldDefault = "[]"
			} else if fieldDefault != "" && (fieldType == "string" || fieldType == "duration") {
				fieldDefault = `"` + fieldDefault + `"`
			}

			a := attr{
				Default:     fieldDefault,
				Description: fieldDescription,
				Name:        name,
				Type:        fieldType,
			}
			result.Attributes = append(result.Attributes, a)
		}

		sort.Sort(byName(result.Attributes))
		if result.Blocks != nil {
			sort.Sort(byName(result.Blocks))
		}

		var bAttr, bBlock *bytes.Buffer

		if result.Attributes != nil {
			bAttr = &bytes.Buffer{}
			enc := json.NewEncoder(bAttr)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			if err := enc.Encode(result.Attributes); err != nil {
				panic(err)
			}
		}

		if result.Blocks != nil {
			bBlock = &bytes.Buffer{}
			enc := json.NewEncoder(bBlock)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			if err := enc.Encode(result.Blocks); err != nil {
				panic(err)
			}
		}

		// TODO: write func
		file, err := os.OpenFile(filepath.Join(docsBlockPath, blockName+".md"), os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}

		fileBytes := &bytes.Buffer{}

		scanner := bufio.NewScanner(file)
		var skipMode, seenAttr, seenBlock bool
		var endToken string
		for scanner.Scan() {
			line := scanner.Text()

			// handle attributes/blocks markers in either legacy (::attributes/::blocks) or shortcode ({{< attributes >}}/{{< blocks >}}) form
			if bAttr != nil && (strings.HasPrefix(line, "::attributes") || strings.HasPrefix(line, "{{< attributes")) {
				// write shortcode version
				fileBytes.WriteString("{{< attributes >}}\n")
				fileBytes.WriteString(bAttr.String())
				fileBytes.WriteString("{{< /attributes >}}\n")
				skipMode = true
				if strings.HasPrefix(line, "::attributes") {
					endToken = "::"
				} else {
					endToken = "{{< /attributes >}}"
				}
				seenAttr = true
				continue
			} else if bBlock != nil && (strings.HasPrefix(line, "::blocks") || strings.HasPrefix(line, "{{< blocks")) {
				// write shortcode version
				fileBytes.WriteString("{{< blocks >}}\n")
				fileBytes.WriteString(bBlock.String())
				fileBytes.WriteString("{{< /blocks >}}\n")
				skipMode = true
				if strings.HasPrefix(line, "::blocks") {
					endToken = "::"
				} else {
					endToken = "{{< /blocks >}}"
				}
				seenBlock = true
				continue
			}

			if skipMode && line == endToken {
				skipMode = false
				endToken = ""
				continue
			}

			if !skipMode {
				fileBytes.Write(scanner.Bytes())
				fileBytes.Write([]byte("\n"))
			}
		}

		if bAttr != nil && !seenAttr {
			fileBytes.WriteString("\n{{< attributes >}}\n")
			fileBytes.WriteString(bAttr.String())
			fileBytes.WriteString("{{< /attributes >}}\n")
		}
		if bBlock != nil && !seenBlock {
			fileBytes.WriteString("\n{{< blocks >}}\n")
			fileBytes.WriteString(bBlock.String())
			fileBytes.WriteString("{{< /blocks >}}\n")
		}

		size, err := file.WriteAt(fileBytes.Bytes(), 0)
		if err != nil {
			panic(err)
		}
		err = os.Truncate(file.Name(), int64(size))
		if err != nil {
			panic(err)
		}

		processedFiles[file.Name()] = struct{}{}
		println("Attributes/Blocks written: "+blockName+":\r\t\t\t\t\t", file.Name())

		// Collect entries for indexing after clearing
		if os.Getenv(searchClientKey) != "" {
			allEntries = append(allEntries, result)
		}
	}

	if os.Getenv(searchClientKey) == "" {
		return
	}

	// Clear existing index before rebuilding - done here after all file generation is complete
	_, err := index.ClearObjects()
	if err != nil {
		panic(err)
	}
	println("SearchIndex cleared - rebuilding...")

	// Save all collected block entries
	if len(allEntries) > 0 {
		_, err = index.SaveObjects(allEntries)
		if err != nil {
			panic(err)
		}
		println("Indexed", len(allEntries), "configuration blocks")
	}

	// Note: Algolia index settings (searchable attributes, ranking, etc.)
	// can be configured via the Algolia dashboard if needed
	// Default settings work well for our use case

	// Index all markdown files recursively in configuration and other sections
	indexDirectoryRecursive(configurationPath, processedFiles, index)

	// Also index getting-started and observation sections
	indexDirectoryRecursive("docs/website/content/getting-started", processedFiles, index)
	indexDirectoryRecursive("docs/website/content/observation", processedFiles, index)
}

func collectFields(t reflect.Type, fields []reflect.StructField) []reflect.StructField {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			fields = append(fields, collectFields(field.Type, fields)...)
		} else {
			fields = append(fields, field)
		}
	}
	return fields
}

var mdHeaderRegex = regexp.MustCompile(`#(.+)\n(\n(.+)\n)`)
var mdFileRegex = regexp.MustCompile(`\d?\.?(.+)\.md`)

func indexDirectory(dirPath, docType string, processedFiles map[string]struct{}, index *search.Index) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		entryPath := filepath.Join(dirPath, dirEntry.Name())
		if _, ok := processedFiles[entryPath]; ok {
			continue
		}

		println("Indexing from file: " + dirEntry.Name())
		fileContent, rerr := os.ReadFile(entryPath)
		if rerr != nil {
			panic(rerr)
		}
		println(dirEntry.Name())
		fileName := mdFileRegex.FindStringSubmatch(dirEntry.Name())[1]
		dt := docType
		if dt == "" {
			dt = fileName
		} else {
			fileName, _ = url.JoinPath(dt, fileName)
		}
		title, description, indexTable := headerFromMeta(fileContent)
		if title == "" && description == "" {
			matches := mdHeaderRegex.FindSubmatch(fileContent)
			description = string(bytes.ToLower(matches[3]))
			title = string(bytes.ToLower(matches[1]))
		}

		urlPath, _ := url.JoinPath(urlBasePath, fileName)
		result := &entry{
			Attributes:  attributesFromTable(fileContent, indexTable),
			Description: description,
			ID:          urlPath,
			Name:        title,
			Type:        dt,
			URL:         urlPath,
		}

		// debug
		if index == nil {
			b, merr := json.Marshal(result)
			if merr != nil {
				panic(merr)
			}
			println(string(b))
		} else {
			_, err = index.SaveObjects(result)
			if err != nil {
				panic(err)
			}
			println("SearchIndex updated")
		}
	}
}

func headerFromMeta(content []byte) (title string, description string, indexTable bool) {
	var metaSep = []byte(`---`)
	if !bytes.HasPrefix(content, metaSep) {
		return
	}
	endIdx := bytes.LastIndex(content, metaSep)
	s := bufio.NewScanner(bytes.NewReader(content[3:endIdx]))
	for s.Scan() {
		t := s.Text()
		if strings.HasPrefix(t, "title") {
			title = strings.Split(t, ": ")[1]
		} else if strings.HasPrefix(t, "description") {
			description = strings.Split(t, ": ")[1]
		} else if strings.HasPrefix(t, "indexTable") {
			indexTable = t == "indexTable: true"
		}

	}
	return
}

var tableEntryRegex = regexp.MustCompile(`^\|\s\x60(.+)\x60\s+\|\s(.+)\s\|\s(.+)\.\s+\|`)

func attributesFromTable(content []byte, parse bool) []interface{} {
	if !parse {
		return nil
	}
	attrs := make([]interface{}, 0)
	s := bufio.NewScanner(bytes.NewReader(content))
	var tableHeadSeen bool
	for s.Scan() {
		// scan to table header
		line := s.Text()
		if !tableHeadSeen {
			if strings.HasPrefix(line, "|:-") {
				tableHeadSeen = true
			}
			continue
		}
		if line[0] != '|' {
			break
		}
		matches := tableEntryRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}
		attrs = append(attrs, attr{
			Description: strings.TrimSpace(matches[3]),
			Name:        strings.TrimSpace(matches[1]),
			Type:        strings.TrimSpace(matches[2]),
		})
	}
	sort.Sort(byName(attrs))
	return attrs
}

type byName []interface{}

func (entries byName) Len() int {
	return len(entries)
}
func (entries byName) Swap(i, j int) {
	entries[i], entries[j] = entries[j], entries[i]
}
func (entries byName) Less(i, j int) bool {
	left := reflect.ValueOf(entries[i]).FieldByName("Name").String()
	right := reflect.ValueOf(entries[j]).FieldByName("Name").String()
	return left < right
}

func newFields(impl interface{}) []reflect.StructField {
	it := reflect.TypeOf(impl).Elem()
	var fields []reflect.StructField
	for i := 0; i < it.NumField(); i++ {
		fields = append(fields, it.Field(i))
	}
	return fields
}

// extractBlockDescription reads the block's markdown file to extract its description
func extractBlockDescription(blockName string) string {
	mdPath := filepath.Join(docsBlockPath, blockName+".md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return ""
	}

	// Extract description - skip frontmatter, skip H1, collect content until table/shortcode
	lines := bytes.Split(content, []byte("\n"))
	var inFrontmatter bool
	var pastH1 bool
	var description strings.Builder
	var emptyLineCount int

	for _, line := range lines {
		lineStr := strings.TrimSpace(string(line))

		// Handle frontmatter
		if lineStr == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}
		if inFrontmatter {
			continue
		}

		// Skip H1 heading
		if strings.HasPrefix(lineStr, "# ") {
			pastH1 = true
			continue
		}

		// Only collect content after H1
		if !pastH1 {
			continue
		}

		// Stop at tables, shortcodes, or comments
		if strings.HasPrefix(lineStr, "|") ||
			strings.HasPrefix(lineStr, "{{<") ||
			strings.HasPrefix(lineStr, "<!--") {
			break
		}

		// Handle empty lines
		if lineStr == "" {
			emptyLineCount++
			// Stop after 2 consecutive empty lines (end of intro section)
			if emptyLineCount >= 2 && description.Len() > 0 {
				break
			}
			continue
		}

		emptyLineCount = 0

		// Handle blockquotes - extract text content
		if strings.HasPrefix(lineStr, ">") {
			blockquoteText := strings.TrimPrefix(lineStr, ">")
			blockquoteText = strings.TrimSpace(blockquoteText)
			// Skip emoji-only or very short content
			if len(blockquoteText) > 5 {
				if description.Len() > 0 {
					description.WriteString(" ")
				}
				description.WriteString(blockquoteText)
			}
			continue
		}

		// Add regular text
		if description.Len() > 0 {
			description.WriteString(" ")
		}
		description.WriteString(lineStr)
	}

	result := description.String()

	// Clean up markdown syntax
	// Remove inline code backticks for cleaner display
	result = strings.ReplaceAll(result, "`", "")
	// Remove links but keep the text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	result = linkRegex.ReplaceAllString(result, "$1")

	// Limit to ~200 characters
	if len(result) > 200 {
		result = result[:197] + "..."
	}

	return result
}

// indexDirectoryRecursive indexes all markdown files in a directory and its subdirectories
func indexDirectoryRecursive(dirPath string, processedFiles map[string]struct{}, index *search.Index) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		println("Warning: could not read directory " + dirPath + ": " + err.Error())
		return
	}

	for _, dirEntry := range dirEntries {
		entryPath := filepath.Join(dirPath, dirEntry.Name())

		if dirEntry.IsDir() {
			// Recursively index subdirectories
			indexDirectoryRecursive(entryPath, processedFiles, index)
			continue
		}

		// Skip non-markdown files and already processed files
		if !strings.HasSuffix(dirEntry.Name(), ".md") || dirEntry.Name() == "_index.md" {
			continue
		}

		if _, ok := processedFiles[entryPath]; ok {
			continue
		}

		println("Indexing file: " + entryPath)

		fileContent, rerr := os.ReadFile(entryPath)
		if rerr != nil {
			println("Warning: could not read file " + entryPath + ": " + rerr.Error())
			continue
		}

		// Extract URL path from file path
		relativePath := strings.TrimPrefix(entryPath, "docs/website/content/")
		relativePath = strings.TrimSuffix(relativePath, ".md")
		// Remove number prefixes from path segments
		pathParts := strings.Split(relativePath, "/")
		for i, part := range pathParts {
			pathParts[i] = strings.TrimPrefix(part, regexp.MustCompile(`^\d+\.`).FindString(part))
		}
		urlPath := "/" + strings.Join(pathParts, "/") + "/"

		// Extract title and description from frontmatter
		title, description, _ := headerFromMeta(fileContent)

		// If no frontmatter, extract from first H1
		if title == "" {
			h1Regex := regexp.MustCompile(`(?m)^#\s+(.+)$`)
			if matches := h1Regex.FindSubmatch(fileContent); len(matches) > 1 {
				title = string(matches[1])
			}
		}

		if title == "" {
			title = filepath.Base(relativePath)
		}

		// Determine document type from path
		docType := "documentation"
		docTypeLabel := "Documentation"
		if strings.Contains(relativePath, "block/") {
			docType = "block"
			docTypeLabel = "Configuration Block"
		} else if strings.Contains(relativePath, "getting-started") {
			docType = "getting-started"
			docTypeLabel = "Getting Started"
		} else if strings.Contains(relativePath, "observation") {
			docType = "observation"
			docTypeLabel = "Observation"
		} else if strings.Contains(relativePath, "configuration") {
			docType = "configuration"
			docTypeLabel = "Configuration"
		}

		// Add source path to description for context
		sourceContext := docTypeLabel + " → " + strings.ReplaceAll(relativePath, "/", " › ")
		if description != "" {
			description = description + " | Source: " + sourceContext
		} else {
			description = "Source: " + sourceContext
		}

		// Create main entry for the page
		result := &entry{
			Description: description,
			ID:          urlPath,
			Name:        title,
			Type:        docType,
			URL:         urlPath,
			Attributes:  extractSearchableContent(fileContent),
		}

		_, err = index.SaveObjects(result)
		if err != nil {
			println("Warning: failed to index " + entryPath + ": " + err.Error())
		} else {
			println("Indexed: " + urlPath)
		}

		// Also index H2 and H3 headings as separate searchable entries
		indexHeadings(fileContent, urlPath, title, index)
	}
}

// extractSearchableContent extracts code blocks, lists, and other searchable content
func extractSearchableContent(content []byte) []interface{} {
	attrs := make([]interface{}, 0)

	// Extract code blocks with context
	codeBlockRegex := regexp.MustCompile("(?s)```\\w*\\n(.*?)```")
	codeMatches := codeBlockRegex.FindAllSubmatch(content, -1)
	for _, match := range codeMatches {
		if len(match) > 1 {
			codeSnippet := string(match[1])
			if len(codeSnippet) > 0 && len(codeSnippet) < 500 {
				attrs = append(attrs, attr{
					Name:        "code",
					Description: codeSnippet,
					Type:        "code",
				})
			}
		}
	}

	// Extract inline code with surrounding context
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	inlineMatches := inlineCodeRegex.FindAllSubmatch(content, -1)
	seen := make(map[string]bool)
	for _, match := range inlineMatches {
		if len(match) > 1 {
			code := string(match[1])
			if !seen[code] && len(code) < 100 {
				attrs = append(attrs, attr{
					Name:        code,
					Description: "Reference to " + code,
					Type:        "reference",
				})
				seen[code] = true
			}
		}
	}

	return attrs
}

// indexHeadings indexes H2 and H3 headings as separate entries for granular search
func indexHeadings(content []byte, baseURL string, pageTitle string, index *search.Index) {
	// Extract H2 headings
	h2Regex := regexp.MustCompile(`(?m)^##\s+(.+)$`)
	h2Matches := h2Regex.FindAllSubmatch(content, -1)

	for _, match := range h2Matches {
		if len(match) > 1 {
			heading := string(match[1])
			// Create anchor ID (simplified, matching Hugo's default)
			anchor := strings.ToLower(heading)
			anchor = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(anchor, "")
			anchor = regexp.MustCompile(`\s+`).ReplaceAllString(anchor, "-")

			headingEntry := &entry{
				ID:          baseURL + "#" + anchor,
				Name:        heading,
				Description: "Section in " + pageTitle,
				Type:        "heading",
				URL:         baseURL + "#" + anchor,
			}

			_, err := index.SaveObjects(headingEntry)
			if err != nil {
				println("Warning: failed to index heading: " + heading)
			}
		}
	}
}
