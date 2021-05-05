package request

type ContextKey uint8

const (
	UID ContextKey = iota
	AccessControls
	BackendName
	BackendURL
	Endpoint
	EndpointKind
	Error
	OpenAPI
	PathParams
	RoundTripName
	RoundTripProxy
	ServerName
	TokenRequest
	TokenRequestRetries
	URLAttribute
	Wildcard
	XFF
)
