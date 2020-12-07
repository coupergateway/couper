package test

import (
	"bytes"
	"text/template"
)

func (h *Helper) NewOpenAPIConf(path string) []byte {
	openapiYAMLTemplate := template.New("openapiYAML")
	_, err := openapiYAMLTemplate.Parse(`openapi: 3.0.1
info:
  title: Test API
  version: "1.0"
paths:
  {{.path}}:
    get:
      responses:
        200:
          description: OK
`)
	h.Must(err)

	openapiYAML := &bytes.Buffer{}
	h.Must(openapiYAMLTemplate.Execute(openapiYAML, map[string]string{"path": path}))
	return openapiYAML.Bytes()
}
