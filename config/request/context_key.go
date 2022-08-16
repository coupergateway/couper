package request

type ContextKey uint8

const (
	ContextType ContextKey = iota
	APIName
	AccessControls
	BackendBytes
	BackendName
	BackendParams
	BackendTokenRequest
	BetaGrantedPermissions
	BetaRequiredPermission
	BufferOptions
	ConfigDryRun
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
	WebsocketsAllowed
	WebsocketsTimeout
	Wildcard
	XFF
)
