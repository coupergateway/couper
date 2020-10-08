package errors

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
	APIReqBodySizeExceeded
)

const (
	AuthorizationRequired Code = 5000 + iota
	AuthorizationFailed
	BasicAuthFailed
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
	FilesRouteNotFound: "FilesRouteNotFound",
	// 4xxx
	APIError:               "API failed",
	APIRouteNotFound:       "API route not found",
	APIConnect:             "API upstream connection error",
	APIReqBodySizeExceeded: "Request body size exceeded",
	// 5xxx
	AuthorizationRequired: "Authorization required",
	AuthorizationFailed:   "Authorization failed",
	BasicAuthFailed:       "Unauthorized",
}

type Code int

func (c Code) Error() string {
	if msg, ok := codes[c]; ok {
		return msg
	}
	return "not implemented"
}
