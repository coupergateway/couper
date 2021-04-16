package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

func TestInline_Backend(t *testing.T) {
	body := testInline_GetBody(t, []byte(`
	server "test" {
		endpoint "/" {
			proxy {
				backend "test" {
					origin = "https://example.com"
				}
			}
		}
	}
	definitions{
		backend "test" {}
	}
	`))

	backend := &Backend{}

	if backend.HCLBody() != nil {
		t.Error("HCLBody() has to be NIL")
	}
	if backend.Reference() != "" {
		t.Error("Reference() has to be an empty string")
	}

	exp := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "basic_auth", Required: false},
			{Name: "hostname", Required: false},
			{Name: "origin", Required: false},
			{Name: "path_prefix", Required: false},
			{Name: "proxy", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema(nil),
	}

	inlineSchema := backend.Schema(true)
	if !reflect.DeepEqual(inlineSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, inlineSchema)
	}

	backendSchema := newBackendSchema(inlineSchema, nil)
	if !reflect.DeepEqual(backendSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, backendSchema)
	}

	internalSchema := backend.Schema(false)
	if !strings.Contains(fmt.Sprintf("%#v", internalSchema), `hcl.AttributeSchema{Name:"timeout", Required:false},`) {
		t.Error("backend.Schema(false) returns an unexpected result")
	}

	inlineSchema.Blocks = append(inlineSchema.Blocks, hcl.BlockHeaderSchema{
		Type:       "backend",
		LabelNames: []string{"backend"},
	})
	exp.Blocks = []hcl.BlockHeaderSchema{{Type: "backend", LabelNames: []string(nil)}}

	backendSchemaTrue := newBackendSchema(inlineSchema, body)
	if !reflect.DeepEqual(backendSchemaTrue, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, backendSchemaTrue)
	}
}

func TestInline_Proxy(t *testing.T) {
	proxy := &Proxy{}

	if proxy.HCLBody() != nil {
		t.Error("HCLBody() has to be NIL")
	}
	if proxy.Reference() != "" {
		t.Error("Reference() has to be an empty string")
	}

	exp := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema(nil),
	}

	internalSchema := proxy.Schema(false)
	if !reflect.DeepEqual(internalSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, internalSchema)
	}

	proxy.BackendName = "backend"
	exp = &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "url", Required: false},
		},
	}

	inlineSchema := proxy.Schema(true)
	if !reflect.DeepEqual(inlineSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, inlineSchema)
	}
}

func TestInline_Request(t *testing.T) {
	request := &Request{}

	if request.HCLBody() != nil {
		t.Error("HCLBody() has to be NIL")
	}
	if request.Reference() != "" {
		t.Error("Reference() has to be an empty string")
	}

	exp := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema(nil),
	}

	internalSchema := request.Schema(false)
	if !reflect.DeepEqual(internalSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, internalSchema)
	}

	request.BackendName = "backend"
	exp = &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "body", Required: false},
			{Name: "form_body", Required: false},
			{Name: "headers", Required: false},
			{Name: "json_body", Required: false},
			{Name: "method", Required: false},
			{Name: "query_params", Required: false},
			{Name: "url", Required: false},
		},
	}

	inlineSchema := request.Schema(true)
	if !reflect.DeepEqual(inlineSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, inlineSchema)
	}
}

func TestInline_Response(t *testing.T) {
	response := &Response{}
	if response.HCLBody() != nil {
		t.Error("HCLBody() has to be NIL")
	}

	exp := &hcl.BodySchema{}

	internalSchema := response.Schema(false)
	if !reflect.DeepEqual(internalSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, internalSchema)
	}
}

func TestInline_Endpoint(t *testing.T) {
	endpoint := &Endpoint{}

	if endpoint.HCLBody() != nil {
		t.Error("HCLBody() has to be NIL")
	}

	exp := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "access_control", Required: false},
			{Name: "disable_access_control", Required: false},
			{Name: "error_file", Required: false},
			{Name: "request_body_limit", Required: false},
		}, Blocks: []hcl.BlockHeaderSchema{
			{Type: "response", LabelNames: []string(nil)},
		},
	}

	internalSchema := endpoint.Schema(false)
	if !reflect.DeepEqual(internalSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, internalSchema)
	}

	exp = &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "proxy", LabelNames: []string{"name"}},
			{Type: "request", LabelNames: []string{"name"}},
		},
	}

	inlineSchema := endpoint.Schema(true)
	if !reflect.DeepEqual(inlineSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, inlineSchema)
	}
}

func TestInline_OAuth2(t *testing.T) {
	oauth2 := &OAuth2{}

	if oauth2.HCLBody() != nil {
		t.Error("HCLBody() has to be NIL")
	}
	if oauth2.Reference() != "" {
		t.Error("Reference() has to be an empty string")
	}

	exp := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: false},
			{Name: "grant_type", Required: true},
		},
	}

	internalSchema := oauth2.Schema(false)
	if !reflect.DeepEqual(internalSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, internalSchema)
	}

	oauth2.BackendName = "backend"
	exp = &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "client_id", Required: true},
			{Name: "client_secret", Required: true},
			{Name: "token_endpoint", Required: false},
		},
	}

	inlineSchema := oauth2.Schema(true)
	if !reflect.DeepEqual(inlineSchema, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, inlineSchema)
	}
}

func testInline_GetBody(t *testing.T, src []byte) hcl.Body {
	parser := hclparse.NewParser()

	file, _ := parser.ParseHCL([]byte(`
	server "test" {
		endpoint "/" {
			proxy {
				backend "test" {
					origin = "https://example.com"
				}
			}
		}
	}
	definitions{
		backend "test" {}
	}
	`), "test.hcl")

	if file == nil || file.Body == nil {
		t.Fatal("Empty file.Body")
	}

	return file.Body
}
