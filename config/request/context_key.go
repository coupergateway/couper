package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	Endpoint
	EndpointKind
	PathParams
	RoundTripProxy
	RoundTripName
	ServerName
	Wildcard
)
