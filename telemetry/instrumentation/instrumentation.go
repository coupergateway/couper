package instrumentation

const (
	Name   = "github.com/avenga/couper/telemetry"
	Prefix = "couper_"

	ClientRequest          = Prefix + "client_request_total"
	ClientRequestDuration  = Prefix + "client_request_duration_seconds"
	BackendRequest         = Prefix + "backend_request_total"
	BackendRequestDuration = Prefix + "backend_request_duration_seconds"
	Connections            = Prefix + "connections_count"
	ConnectionsTotal       = Prefix + "connections_total"
	ConnectionsLifetime    = Prefix + "connections_lifetime_seconds"
)
