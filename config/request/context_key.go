package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
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
	UID
	URLAttribute
	WebsocketsTimeout
	Wildcard
	XFF
)
