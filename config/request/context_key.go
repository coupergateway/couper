package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	ContextVariablesSynced
	APIName
	AccessControls
	BackendName
	BufferOptions
	Endpoint
	EndpointExpectedStatus
	EndpointKind
	Error
	Handler
	LogCustomAccess
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
