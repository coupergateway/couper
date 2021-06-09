package request

type ContextKey uint8

const (
	UID ContextKey = iota
	AccessControls
	BackendName
	Endpoint
	EndpointKind
	Error
	LogEntry
	OpenAPI
	PathParams
	ResponseBodyLen
	RoundTripName
	RoundTripProxy
	ServerName
	TokenRequest
	TokenRequestRetries
	URLAttribute
	Wildcard
	XFF
)
