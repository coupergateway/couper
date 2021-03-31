package request

type ContextKey uint8

const (
	UID ContextKey = iota
	AccessControl
	AccessControls
	BackendName
	BackendURL
	Endpoint
	EndpointKind
	OpenAPI
	PathParams
	RoundTripName
	RoundTripProxy
	ServerName
	TokenRequest
	URLAttribute
	Wildcard
	XFF
)
