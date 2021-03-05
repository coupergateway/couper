package request

type ContextKey uint8

const (
	UID ContextKey = iota
	AccessControls
	BackendName
	Endpoint
	EndpointKind
	IsResourceReq
	OpenAPI
	MemStore
	PathParams
	RoundTripName
	RoundTripProxy
	ServerName
	TokenEndpoint
	TokenKey
	Wildcard
	XFF
)
