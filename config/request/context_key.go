package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	APIName
	AccessControls
	BackendName
	BufferOptions
	Endpoint
	EndpointKind
	Error
	Handler
	LogCustomAccess
	LogCustomEvalResult
	LogCustomUpstream
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
