package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	DisableLogging
	Endpoint
	MemStore
	PathParams
	RoundtripInfo
	SendAuthHeader
	ServerName
	Wildcard
)
