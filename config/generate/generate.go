package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
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

const docsBlockPath = "docs/website/content/2.configuration/4.block"

// export md: 1) search for ::attribute, replace if exist or append at end
func main() {
	const basePath = "/configuration/block/"

	client := search.NewClient("MSIN2HU7WH", os.Getenv("SEARCH_CLIENT_API_KEY"))
	index := client.InitIndex("docs")

	for _, impl := range []interface{}{
		&config.API{},
		&config.Backend{},
		&config.BasicAuth{},
		&config.CORS{},
		&config.Defaults{},
		&config.Files{},
		&config.Proxy{},
		&config.Endpoint{},
	} {
		t := reflect.TypeOf(impl).Elem()
		name := strings.TrimPrefix(strings.ToLower(fmt.Sprintf("%v", t)), "config.")
		fileName := errors.TypeToSnake(impl)

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
				} else if ft != "string" {
					ft = "object"
				}
				fieldType = ft
			}

			a := attr{
				Name:        strings.Split(field.Tag.Get("hcl"), ",")[0],
				Type:        fieldType,
				Description: field.Tag.Get("docs"),
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
		//continue
		_, err = index.SaveObjects(result) //, opt.AutoGenerateObjectIDIfNotExist(true))
		if err != nil {
			panic(err)
		}
		println("SearchIndex updated")
	}
}