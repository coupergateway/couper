package request

type ContextKey uint8

const (
	UID ContextKey = iota
	AccessControls
	AllowWebsockets
	BackendName
	Endpoint
	EndpointKind
	Error
	LogEntry
	OpenAPI
	PathParams
	RoundTripName
	RoundTripProxy
	RW
	ServerName
	TokenRequest
	TokenRequestRetries
	URLAttribute
	Wildcard
	XFF
)
