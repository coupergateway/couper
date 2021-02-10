package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	ConfigKey
	Endpoint
	IsResourceReq
	MemStore
	PathParams
	RoundtripInfo
	ServerName
	SourceRequest
	StartTime
	Wildcard
)
