package request

type ContextKey uint8

const (
	UID ContextKey = iota
	BackendName
	Endpoint
	EndpointKind
	OpenAPI
	PathParams
	RoundTripName
	RoundTripProxy
	ServerName
	Wildcard
)
