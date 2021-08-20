package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	AccessControls
	BackendName
	Endpoint
	EndpointKind
	Error
	LogEntry
	OpenAPI
	PathParams
	RoundTripName
	RoundTripProxy
	ServerName
	TokenRequest
	TokenRequestRetries
	UID
	URLAttribute
	Wildcard
	XFF
)
