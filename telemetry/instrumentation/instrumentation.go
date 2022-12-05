package instrumentation

const (
	Name   = "github.com/avenga/couper/telemetry"
	Prefix = "couper_"

	BackendInstrumentationName = "couper/backend"

	BackendConnections         = Prefix + "backend_connections_count"
	BackendConnectionsLifetime = Prefix + "backend_connections_lifetime_seconds"
	BackendConnectionsTotal    = Prefix + "backend_connections"
	BackendHealthState         = Prefix + "backend_up"
	BackendRequest             = Prefix + "backend_request"
	BackendRequestDuration     = Prefix + "backend_request_duration_seconds"
	ClientConnections          = Prefix + "client_connections_count"
	ClientConnectionsTotal     = Prefix + "client_connections"
	ClientRequest              = Prefix + "client_request"
	ClientRequestDuration      = Prefix + "client_request_duration_seconds"
)
