package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	ConfigKey
	Endpoint
	MemStore
	PathParams
	RoundtripInfo
	SendAuthHeader
	ServerName
	SourceRequest
	StartTime
	Wildcard
)
