package errors

import "net/http"

// Definitions holds all implemented ones. The name must match the structs
// snake-name for fallback purposes. See TypeToSnake usage and reference.
var Definitions = []*Error{
	AccessControl.Kind("basic_auth").Status(http.StatusUnauthorized),
	AccessControl.Kind("basic_auth").Kind("basic_auth_credentials_missing").Status(http.StatusUnauthorized),

	AccessControl.Kind("jwt"),
	AccessControl.Kind("jwt").Kind("jwt_token_expired"),
	AccessControl.Kind("jwt").Kind("jwt_token_missing").Status(http.StatusUnauthorized),
	AccessControl.Kind("jwt").Kind("jwt_token_invalid"),

	AccessControl.Kind("saml2"),

	AccessControl.Kind("oauth2"),
}
