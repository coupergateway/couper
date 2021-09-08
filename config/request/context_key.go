package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	AccessControls
	BackendName
	Endpoint
	EndpointKind
	Error
	Handler
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
	WebsocketsAllowed
	WebsocketsTimeout
	Wildcard
	XFF
)
