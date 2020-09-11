package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	Endpoint
	ServerName
	RoundtripInfo
	Wildcard
)
