package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	Endpoint
	MemStore
	PathParams
	RoundtripInfo
	SendAuthHeader
	ServerName
	Wildcard
)
