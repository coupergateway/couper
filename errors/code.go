package errors

const (
	ServerError Code = 1000 + iota
	ConfigurationError
	RequestError
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
)

const (
	AuthorizationRequired = 5000 + iota
	AuthorizationFailed
)

var codes = map[Code]string{
	// 1xxx
	ServerError:        "Server error",
	ConfigurationError: "Configuration failed",
	RequestError:       "Invalid request",
	RouteNotFound:      "Route not found",
	// 2xxx
	SPAError:         "SPA failed",
	SPARouteNotFound: "SPA route not found",
	// 3xxx
	FilesError:         "Files failed",
	FilesRouteNotFound: "FilesRouteNotFound",
	// 4xxx
	APIError:         "API failed",
	APIRouteNotFound: "API route not found",
	// 5xxx
	AuthorizationRequired: "Authorization required",
	AuthorizationFailed:   "Authorization failed",
}

type Code int

func (c Code) Error() string {
	if msg, ok := codes[c]; ok {
		return msg
	}
	return "not implemented"
}
