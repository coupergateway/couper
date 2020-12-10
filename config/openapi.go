package config

type OpenAPI struct {
	File                     string `hcl:"file"`
	ExcludeRequestBody       bool   `hcl:"exclude_request_body,optional"`
	ExcludeResponseBody      bool   `hcl:"exclude_response_body,optional"`
	ExcludeStatusCode        bool   `hcl:"exclude_status_code,optional"`
	IgnoreRequestViolations  bool   `hcl:"ignore_request_violations,optional"`
	IgnoreResponseViolations bool   `hcl:"ignore_response_violations,optional"`
}
