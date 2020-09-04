package request

type ContextKey uint8

const (
	UID ContextKey = iota
	Endpoint
	ServerName
	RoundtripInfo
	Wildcard
)
