package request

type ContextKey uint8

const (
	RequestID ContextKey = iota
	ConnectionSerial
	Endpoint
	Wildcard
)
