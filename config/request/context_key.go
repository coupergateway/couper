package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	APIName
	AccessControls
	BackendBytes
	BackendName
	BackendParams
	BufferOptions
	ConfigDryRun
	ConnectTimeout
	ContextVariablesSynced
	Endpoint
	EndpointExpectedStatus
	EndpointKind
	EndpointSequenceDependsOn
	Error
	GrantedPermissions
	Handler
	LogCustomAccess
	LogCustomUpstreamValue
	LogCustomUpstreamError
	LogDebugLevel
	LogEntry
	OpenAPI
	PathParams
	RequiredPermission
	ResponseBlock
	ResponseWriter
	RoundTripName
	RoundTripProxy
	ServerName
	ServerTimings
	StartTime
	TokenRequest
	TokenRequestRetries
	UID
	WebsocketsAllowed
	WebsocketsTimeout
	Wildcard
	XFF
)
