package errors

// typeDefinitions holds all related error definitions which are
// catchable with an error_handler definition.
type typeDefinitions map[string]*Error

// Types holds all implemented ones. The name must match the structs
// snake-name for fallback purposes. See TypeToSnake usage and reference.
var Types = typeDefinitions{
	"basic_auth":                     AccessControl.Kind("basic_auth"),
	"basic_auth_missing_credentials": AccessControl.Kind("basic_auth").Kind("basic_auth_missing_credentials"),

	"jwt":                AccessControl.Kind("jwt"),
	"jwt_token_expired":  AccessControl.Kind("jwt").Kind("jwt_token_expired"),
	"jwt_token_required": AccessControl.Kind("jwt").Kind("jwt_token_required"),
	"jwt_claims":         AccessControl.Kind("jwt").Kind("jwt_claims_invalid"),

	"saml2": AccessControl.Kind("saml2"),
}

// IsKnown tells the configuration callee if Couper
// has a defined error type with the given name.
func (t typeDefinitions) IsKnown(errorType string) bool {
	_, known := t[errorType]
	return known
}
