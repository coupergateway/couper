package config

// OpenAPI represents the <OpenAPI> object.
type OpenAPI struct {
	File                     string `hcl:"file" docs:"OpenAPI YAML definition file"`
	IgnoreRequestViolations  bool   `hcl:"ignore_request_violations,optional" docs:"logs request validation results, skips error handling"`
	IgnoreResponseViolations bool   `hcl:"ignore_response_violations,optional" docs:"logs response validation results, skips error handling"`
}
