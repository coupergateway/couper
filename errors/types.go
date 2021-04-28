package errors

// typeDefinitions holds all related error definitions which are
// catchable with an error_handler definition.
type typeDefinitions map[string]*Error

// Types holds all implemented ones. The name must match the structs
// snake-name for fallback purposes. See TypeToSnake usage and reference.
var Types = typeDefinitions{
	"basic_auth":                     AccessControl.Kind("basic_auth"),
	"basic_auth_missing_credentials": AccessControl.Kind("basic_auth").Kind("basic_auth_missing_credentials"),

	"jwt":                      AccessControl.Kind("jwt"),
	"jwt_token_required":       AccessControl.Kind("jwt_token_required"),
	"jwt_claims_invalid":       AccessControl.Kind("jwt_claims_invalid"),
	"jwt_claims_required":      AccessControl.Kind("jwt_claims_required"),
	"jwt_claims_invalid_value": AccessControl.Kind("jwt_claims_invalid_value"),

	"saml2":                   AccessControl.Kind("saml2"),
	"saml2_audience_required": AccessControl.Kind("saml2_audience_required"),
}

// IsKnown tells the configuration callee if Couper
// has a defined error type with the given name.
func (t typeDefinitions) IsKnown(errorType string) bool {
	_, known := t[errorType]
	return known
}
