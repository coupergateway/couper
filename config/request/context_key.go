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
	ResponseWriter
	RoundTripName
	RoundTripProxy
	ServerName
	TokenRequest
	TokenRequestRetries
	URLAttribute
	WebsocketsTimeout
	Wildcard
	XFF
)
