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
	ErrorKind
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
