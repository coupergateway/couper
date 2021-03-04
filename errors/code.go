package errors

import "fmt"

const (
	Server Code = 1000 + iota
	ServerShutdown
	Configuration
	InvalidRequest
	RouteNotFound
)

const (
	SPAError Code = 2000 + iota
	SPARouteNotFound
)

const (
	FilesError Code = 3000 + iota
	FilesRouteNotFound
)

const (
	APIError Code = 4000 + iota
	APIRouteNotFound
	APIConnect
	APIProxyConnect
)

const (
	AuthorizationRequired Code = 5000 + iota
	AuthorizationFailed
	BasicAuthFailed
)

const (
	UpstreamRequestValidationFailed Code = 6000 + iota
	UpstreamResponseValidationFailed
	UpstreamResponseBufferingFailed
)

const (
	EndpointError Code = 7000 + iota
	EndpointConnect
	EndpointProxyConnect
	EndpointReqBodySizeExceeded
)

var codes = map[Code]string{
	// 1xxx
	Server:         "Server error",
	ServerShutdown: "Server is shutting down",
	Configuration:  "Configuration failed",
	InvalidRequest: "Invalid request",
	RouteNotFound:  "Route not found",
	// 2xxx
	SPAError:         "SPA failed",
	SPARouteNotFound: "SPA route not found",
	// 3xxx
	FilesError:         "Files failed",
	FilesRouteNotFound: "Files route not found",
	// 4xxx
	APIError:         "API failed",
	APIRouteNotFound: "API route not found",
	APIConnect:       "API upstream connection error",
	APIProxyConnect:  "upstream connection error via configured proxy",
	// 5xxx
	AuthorizationRequired: "Authorization required",
	AuthorizationFailed:   "Authorization failed",
	BasicAuthFailed:       "Unauthorized",
	// 6xxx
	UpstreamRequestValidationFailed:  "Upstream request validation failed",
	UpstreamResponseValidationFailed: "Upstream response validation failed",
	UpstreamResponseBufferingFailed:  "Upstream response buffering failed",
	// 7xxx
	EndpointConnect:             "Endpoint upstream connection error",
	EndpointProxyConnect:        "upstream connection error via configured proxy",
	EndpointReqBodySizeExceeded: "Request body size exceeded",
}

type Code int

// TODO: Own error type
// New creates a standard error.
func New(msg string) error {
	return fmt.Errorf(msg)
}

func (c Code) Error() string {
	if msg, ok := codes[c]; ok {
		return msg
	}
	return "not implemented"
}
