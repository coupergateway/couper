package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	APIName
	AccessControls
	BackendName
	BackendParams
	BufferOptions
	ConnectTimeout
	ContextVariablesSynced
	Endpoint
	EndpointExpectedStatus
	EndpointKind
	EndpointSequenceDependsOn
	Error
	Handler
	LogCustomAccess
	LogCustomUpstream
	LogDebugLevel
	LogEntry
	OpenAPI
	PathParams
	ResponseBlock
	ResponseWriter
	RoundTripName
	RoundTripProxy
	Scopes
	ServerName
	StartTime
	TokenRequest
	TokenRequestRetries
	UID
	WebsocketsAllowed
	WebsocketsTimeout
	Wildcard
	XFF
)
