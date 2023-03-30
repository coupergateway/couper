package eval

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestMustBuffer(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   BufferOption
	}{
		{"no buffer", `endpoint "/" {}`, BufferNone},
		{"buffer req/resp with openapi block", `openapi { file = "test.yaml" }`, BufferResponse},
		{"buffer with context reference", `endpoint "/" { set_response_headers = { x = request.context } }`, BufferRequest},
		{"buffer with nested context reference", `endpoint "/" { set_response_headers = { x = request.context.foo } }`, BufferRequest},
		{"buffer request", `endpoint "/" { set_response_headers = { x = request } }`, BufferRequest},
		{"buffer request body", `endpoint "/" { set_response_headers = { x = request.body } }`, BufferRequest},
		{"buffer request form_body", `endpoint "/" { set_response_headers = { x = request.form_body } }`, BufferRequest},
		{"buffer request json_body", `endpoint "/" { set_response_headers = { x = request.json_body } }`, BufferRequest | JSONParseRequest},
		{"buffer backend_requests specific", `endpoint "/" { set_response_headers = { x = backend_requests.r } }`, BufferRequest},
		{"buffer backend_requests body", `endpoint "/" { set_response_headers = { x = backend_requests.r.body } }`, BufferRequest},
		{"buffer backend_requests form_body", `endpoint "/" { set_response_headers = { x = backend_requests.r.form_body } }`, BufferRequest},
		{"buffer backend_requests json_body", `endpoint "/" { set_response_headers = { x = backend_requests.r.json_body } }`, BufferRequest | JSONParseRequest},
		{"buffer backend_request body", `backend "b" { set_response_headers = { x = backend_request.body } }`, BufferRequest},
		{"buffer backend_request form_body", `backend "b" { set_response_headers = { x = backend_request.form_body } }`, BufferRequest},
		{"buffer backend_request json_body", `backend "b" { set_response_headers = { x = backend_request.json_body } }`, BufferRequest | JSONParseRequest},
		{"buffer request add_form_params", `endpoint "/" { add_form_params = [] }`, BufferRequest},
		{"buffer request set_form_params", `endpoint "/" { set_form_params = [] }`, BufferRequest},
		{"buffer request remove_form_params", `endpoint "/" { remove_form_params = [] }`, BufferRequest},
		{"buffer backend_responses", `endpoint "/" { set_response_headers = { x = backend_responses } }`, BufferResponse},
		{"buffer backend_responses default", `endpoint "/" { set_response_headers = { x = backend_responses.default } }`, BufferResponse},
		{"buffer backend_responses body", `endpoint "/" { set_response_headers = { x = backend_responses.default.body } }`, BufferResponse},
		{"buffer backend_responses json_body", `endpoint "/" { set_response_headers = { x = backend_responses.default.json_body } }`, BufferResponse | JSONParseResponse},
		{"buffer backend_response body", `backend "b" { set_response_headers = { x = backend_response.body } }`, BufferResponse},
		{"buffer backend_response json_body", `backend "b" { set_response_headers = { x = backend_response.json_body } }`, BufferResponse | JSONParseResponse},
		{"buffer request/response", `endpoint "/" {
	set_response_headers = {
	  x = request
	  y = backend_responses
	}
}`, BufferRequest | BufferResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.config), "test.hcl", hcl.InitialPos)
			if diags.HasErrors() {
				st.Error(diags)
			}

			if got := MustBuffer(file.Body); got != tt.want {
				t.Errorf("%s: got: %v, want %v", tt.name, got.GoString(), tt.want.GoString())
			}
		})
	}
}
