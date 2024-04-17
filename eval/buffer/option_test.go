package buffer

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestMustBuffer(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   Option
	}{
		{"no buffer", `endpoint "/" {}`, None},
		{"buffer req/resp with openapi block", `openapi { file = "test.yaml" }`, Response},
		{"buffer with context reference", `endpoint "/" { set_response_headers = { x = request.context } }`, Request},
		{"buffer with nested context reference", `endpoint "/" { set_response_headers = { x = request.context.foo } }`, Request},
		{"buffer request", `endpoint "/" { set_response_headers = { x = request } }`, Request},
		{"buffer request body", `endpoint "/" { set_response_headers = { x = request.body } }`, Request},
		{"buffer request form_body", `endpoint "/" { set_response_headers = { x = request.form_body } }`, Request},
		{"buffer request json_body", `endpoint "/" { set_response_headers = { x = request.json_body } }`, Request | JSONParseRequest},
		{"buffer backend_requests specific", `endpoint "/" { set_response_headers = { x = backend_requests.r } }`, Request},
		{"buffer backend_requests body", `endpoint "/" { set_response_headers = { x = backend_requests.r.body } }`, Request},
		{"buffer backend_requests form_body", `endpoint "/" { set_response_headers = { x = backend_requests.r.form_body } }`, Request},
		{"buffer backend_requests json_body", `endpoint "/" { set_response_headers = { x = backend_requests.r.json_body } }`, Request | JSONParseRequest},
		{"buffer backend_request body", `backend "b" { set_response_headers = { x = backend_request.body } }`, Request},
		{"buffer backend_request form_body", `backend "b" { set_response_headers = { x = backend_request.form_body } }`, Request},
		{"buffer backend_request json_body", `backend "b" { set_response_headers = { x = backend_request.json_body } }`, Request | JSONParseRequest},
		{"buffer request add_form_params", `endpoint "/" { add_form_params = [] }`, Request},
		{"buffer request set_form_params", `endpoint "/" { set_form_params = [] }`, Request},
		{"buffer request remove_form_params", `endpoint "/" { remove_form_params = [] }`, Request},
		{"buffer backend_responses", `endpoint "/" { set_response_headers = { x = backend_responses } }`, Response},
		{"buffer backend_responses default", `endpoint "/" { set_response_headers = { x = backend_responses.default } }`, Response},
		{"buffer backend_responses body", `endpoint "/" { set_response_headers = { x = backend_responses.default.body } }`, Response},
		{"buffer backend_responses json_body", `endpoint "/" { set_response_headers = { x = backend_responses.default.json_body } }`, Response | JSONParseResponse},
		{"buffer backend_response body", `backend "b" { set_response_headers = { x = backend_response.body } }`, Response},
		{"buffer backend_response json_body", `backend "b" { set_response_headers = { x = backend_response.json_body } }`, Response | JSONParseResponse},
		{"buffer request/response", `endpoint "/" {
	set_response_headers = {
	  x = request
	  y = backend_responses
	}
}`, Request | Response},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.config), "test.hcl", hcl.InitialPos)
			if diags.HasErrors() {
				st.Error(diags)
			}

			if got := Must(file.Body); got != tt.want {
				st.Errorf("%s: got: %v, want %v", tt.name, got.GoString(), tt.want.GoString())
			}
		})
	}
}
