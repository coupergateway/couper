package errors

import "net/http"

// Definitions holds all implemented ones. The name must match the structs
// snake-name for fallback purposes. See TypeToSnake usage and reference.
var Definitions = []*Error{
	AccessControl,

	AccessControl.Kind("basic_auth").Status(http.StatusUnauthorized),
	AccessControl.Kind("basic_auth").Kind("basic_auth_credentials_missing").Status(http.StatusUnauthorized),

	AccessControl.Kind("jwt"),
	AccessControl.Kind("jwt").Kind("jwt_token_expired"),
	AccessControl.Kind("jwt").Kind("jwt_token_invalid").Status(http.StatusUnauthorized),
	AccessControl.Kind("jwt").Kind("jwt_token_missing").Status(http.StatusUnauthorized),

	AccessControl.Kind("oauth2"),

	AccessControl.Kind("saml2"),
	AccessControl.Kind("saml2").Kind("saml"),

	AccessControl.Kind("insufficient_permissions").Context("api").Context("endpoint"),

	Backend,
	Backend.Kind("backend_openapi_validation").Status(http.StatusBadRequest),
	Backend.Kind("beta_backend_rate_limit_exceeded").Status(http.StatusTooManyRequests),
	Backend.Kind("backend_timeout").Status(http.StatusGatewayTimeout),
	Backend.Kind("beta_backend_token_request"),
	Backend.Kind("backend_unhealthy"),

	Endpoint,
	Endpoint.Kind("sequence"),
	Endpoint.Kind("unexpected_status"),
}
