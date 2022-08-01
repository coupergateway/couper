//go:build exclude

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"

	"github.com/avenga/couper/config"
)

type entry struct {
	ID         string `json:"objectID"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Url        string `json:"url"`
	Attributes []attr `json:"attributes"`
}

type attr struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

const (
	searchAppID     = "MSIN2HU7WH"
	searchIndex     = "docs"
	searchClientKey = "SEARCH_CLIENT_API_KEY"

	docsBlockPath = "docs/website/content/2.configuration/4.block"
)

// export md: 1) search for ::attribute, replace if exist or append at end
func main() {
	const basePath = "/configuration/block/"

	client := search.NewClient(searchAppID, os.Getenv(searchClientKey))
	index := client.InitIndex(searchIndex)

	filenameRegex := regexp.MustCompile(`(URL|JWT|OpenAPI|[a-z]+)`)
	bracesRegex := regexp.MustCompile(`{([^}]*)}`)

	for _, impl := range []interface{}{
		&config.API{},
		&config.Backend{},
		&config.BasicAuth{},
		&config.CORS{},
		&config.Defaults{},
		&config.Endpoint{},
		&config.Files{},
		&config.Health{},
		&config.JWTSigningProfile{},
		&config.OpenAPI{},
		&config.Proxy{},
		&config.Request{},
	} {
		t := reflect.TypeOf(impl).Elem()
		name := reflect.TypeOf(impl).String()
		name = strings.TrimPrefix(name, "*config.")
		fileName := strings.ToLower(strings.Trim(filenameRegex.ReplaceAllString(name, "${1}_"), "_"))

		result := entry{
			Name: name,
			Url:  basePath + name,
			Type: "block",
		}
		result.ID = result.Url

		var fields []reflect.StructField
		for i := 0; i < t.NumField(); i++ {
			fields = append(fields, t.Field(i))
		}

		inlineType, ok := impl.(config.Inline)
		if ok {
			it := reflect.TypeOf(inlineType.Inline()).Elem()
			for i := 0; i < it.NumField(); i++ {
				fields = append(fields, it.Field(i))
			}
		}

		for _, field := range fields {
			if field.Tag.Get("docs") == "" {
				continue
			}

			fieldType := field.Tag.Get("type")
			if fieldType == "" {
				ft := field.Type.String()
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
			} else if fieldType == "string" || fieldType == "duration" {
				fieldDefault = `"` + fieldDefault + `"`
			}

			fieldDescription := field.Tag.Get("docs")
			fieldDescription = bracesRegex.ReplaceAllString(fieldDescription, "`${1}`")

			a := attr{
				Default:     fieldDefault,
				Description: fieldDescription,
				Name:        strings.Split(field.Tag.Get("hcl"), ",")[0],
				Type:        fieldType,
			}
			result.Attributes = append(result.Attributes, a)
		}

		b := &bytes.Buffer{}
		enc := json.NewEncoder(b)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result.Attributes); err != nil {
			panic(err)
		}

		// TODO: write func
		file, err := os.OpenFile(filepath.Join(docsBlockPath, fileName+".md"), os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}

		fileBytes := &bytes.Buffer{}

		scanner := bufio.NewScanner(file)
		var skipMode, seen bool
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "::attributes") {
				fileBytes.WriteString(fmt.Sprintf(`::attributes
---
values: %s
---
::
`, b.String()))
				skipMode = true
				seen = true
				continue
			}

			if line == "::" {
				skipMode = false
				continue
			}

			if !skipMode {
				fileBytes.Write(scanner.Bytes())
				fileBytes.Write([]byte("\n"))
			}
		}

		if !seen { // TODO: from func/template
			fileBytes.WriteString(fmt.Sprintf(`
::attributes
---
values: %s
---
::
`, b.String()))
		}

		_, err = file.WriteAt(fileBytes.Bytes(), 0)
		if err != nil {
			panic(err)
		}
		println("Attributes written: " + fileName)

		if os.Getenv(searchClientKey) != "" {
			_, err = index.SaveObjects(result) //, opt.AutoGenerateObjectIDIfNotExist(true))
			if err != nil {
				panic(err)
			}
			println("SearchIndex updated")
		}
	}
}
