package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	AccessControls
	APIName
	BackendName
	BufferOptions
	Endpoint
	EndpointKind
	Error
	Handler
	LogDebugLevel
	LogEntry
	OpenAPI
	PathParams
	ResponseWriter
	RoundTripName
	RoundTripProxy
	Scopes
	ServerName
	StartTime
	TokenRequest
	TokenRequestRetries
	UID
	URLAttribute
	WebsocketsAllowed
	WebsocketsTimeout
	Wildcard
	XFF
)
