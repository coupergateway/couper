package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	APIName
	AccessControls
	BackendName
	BackendParams
	BackendTokenRequest
	BetaGrantedPermissions
	BetaRequiredPermission
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
