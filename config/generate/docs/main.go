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

	"github.com/coupergateway/couper/config/generate/shared"
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

func main() {
	client := search.NewClient(searchAppID, os.Getenv(searchClientKey))
	index := client.InitIndex(searchIndex)

	bracesRegex := regexp.MustCompile(`{([^}]*)}`)

	processedFiles := make(map[string]struct{})
	var allEntries []entry

	for _, info := range shared.GetAllConfigStructs() {
		blockName := info.BlockName
		urlPath, _ := url.JoinPath(urlBasePath, "block", blockName)

		blockDescription := extractBlockDescription(blockName)

		result := entry{
			Name:        blockName,
			URL:         strings.ToLower(urlPath),
			Type:        "block",
			Description: blockDescription,
		}

		result.ID = result.URL

		fields := shared.GetInlineFields(info.Impl)

		for _, field := range fields {
			if field.Tag.Get("docs") == "" {
				continue
			}

			hclInfo := shared.ParseHCLTag(field.Tag.Get("hcl"))
			if hclInfo.Name == "" {
				continue
			}

			name := hclInfo.Name
			fieldDescription := field.Tag.Get("docs")
			fieldDescription = bracesRegex.ReplaceAllString(fieldDescription, "`${1}`")

			if hclInfo.IsBlock {
				b := block{
					Description: fieldDescription,
					Name:        name,
				}
				result.Blocks = append(result.Blocks, b)
				continue
			}

			fieldType := field.Tag.Get("type")
			if fieldType == "" {
				fieldType = goTypeToDocType(field.Type)
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

			if bAttr != nil && (strings.HasPrefix(line, "::attributes") || strings.HasPrefix(line, "{{< attributes")) {
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

		if os.Getenv(searchClientKey) != "" {
			allEntries = append(allEntries, result)
		}
	}

	if os.Getenv(searchClientKey) == "" {
		return
	}

	_, err := index.ClearObjects()
	if err != nil {
		panic(err)
	}
	println("SearchIndex cleared - rebuilding...")

	if len(allEntries) > 0 {
		_, err = index.SaveObjects(allEntries)
		if err != nil {
			panic(err)
		}
		println("Indexed", len(allEntries), "configuration blocks")
	}

	indexDirectoryRecursive(configurationPath, processedFiles, index)
	indexDirectoryRecursive("docs/website/content/getting-started", processedFiles, index)
	indexDirectoryRecursive("docs/website/content/observation", processedFiles, index)
}

func goTypeToDocType(t reflect.Type) string {
	ft := strings.Replace(t.String(), "*", "", 1)
	if ft == "config.List" {
		ft = "[]string"
	}
	if len(ft) >= 2 && ft[:2] == "[]" {
		ft = "tuple (" + ft[2:] + ")"
	} else if strings.Contains(ft, "int") {
		ft = "number"
	} else if ft != "string" && ft != "bool" {
		ft = "object"
	}
	return ft
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

func extractBlockDescription(blockName string) string {
	mdPath := filepath.Join(docsBlockPath, blockName+".md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return ""
	}

	lines := bytes.Split(content, []byte("\n"))
	var inFrontmatter bool
	var pastH1 bool
	var description strings.Builder
	var emptyLineCount int

	for _, line := range lines {
		lineStr := strings.TrimSpace(string(line))

		if lineStr == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}
		if inFrontmatter {
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
			strings.HasPrefix(lineStr, "<!--") {
			break
		}

		if lineStr == "" {
			emptyLineCount++
			if emptyLineCount >= 2 && description.Len() > 0 {
				break
			}
			continue
		}

		emptyLineCount = 0

		if strings.HasPrefix(lineStr, ">") {
			blockquoteText := strings.TrimPrefix(lineStr, ">")
			blockquoteText = strings.TrimSpace(blockquoteText)
			if len(blockquoteText) > 5 {
				if description.Len() > 0 {
					description.WriteString(" ")
				}
				description.WriteString(blockquoteText)
			}
			continue
		}

		if description.Len() > 0 {
			description.WriteString(" ")
		}
		description.WriteString(lineStr)
	}

	result := description.String()

	result = strings.ReplaceAll(result, "`", "")
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	result = linkRegex.ReplaceAllString(result, "$1")

	if len(result) > 200 {
		result = result[:197] + "..."
	}

	return result
}

func indexDirectoryRecursive(dirPath string, processedFiles map[string]struct{}, index *search.Index) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		println("Warning: could not read directory " + dirPath + ": " + err.Error())
		return
	}

	for _, dirEntry := range dirEntries {
		entryPath := filepath.Join(dirPath, dirEntry.Name())

		if dirEntry.IsDir() {
			indexDirectoryRecursive(entryPath, processedFiles, index)
			continue
		}

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

		relativePath := strings.TrimPrefix(entryPath, "docs/website/content/")
		relativePath = strings.TrimSuffix(relativePath, ".md")
		pathParts := strings.Split(relativePath, "/")
		for i, part := range pathParts {
			pathParts[i] = strings.TrimPrefix(part, regexp.MustCompile(`^\d+\.`).FindString(part))
		}
		urlPath := "/" + strings.Join(pathParts, "/") + "/"

		title, description, _ := headerFromMeta(fileContent)

		if title == "" {
			h1Regex := regexp.MustCompile(`(?m)^#\s+(.+)$`)
			if matches := h1Regex.FindSubmatch(fileContent); len(matches) > 1 {
				title = string(matches[1])
			}
		}

		if title == "" {
			title = filepath.Base(relativePath)
		}

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

		sourceContext := docTypeLabel + " → " + strings.ReplaceAll(relativePath, "/", " › ")
		if description != "" {
			description = description + " | Source: " + sourceContext
		} else {
			description = "Source: " + sourceContext
		}

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

		indexHeadings(fileContent, urlPath, title, index)
	}
}

func extractSearchableContent(content []byte) []interface{} {
	attrs := make([]interface{}, 0)

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

func indexHeadings(content []byte, baseURL string, pageTitle string, index *search.Index) {
	h2Regex := regexp.MustCompile(`(?m)^##\s+(.+)$`)
	h2Matches := h2Regex.FindAllSubmatch(content, -1)

	for _, match := range h2Matches {
		if len(match) > 1 {
			heading := string(match[1])
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
