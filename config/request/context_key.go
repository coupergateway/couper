package request

type ContextKey uint8

const (
	UID ContextKey = iota
	ConnectionSerial
	Endpoint
	StartTime
	RoundtripInfo
	Wildcard
)
