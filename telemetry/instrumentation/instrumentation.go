package instrumentation

const (
	Name   = "github.com/avenga/couper/telemetry"
	Prefix = "couper_"

	BackendConnections         = Prefix + "backend_connections_count"
	BackendConnectionsLifetime = Prefix + "backend_connections_lifetime_seconds"
	BackendConnectionsTotal    = Prefix + "backend_connections_total"
	BackendHealthState         = Prefix + "backend_healthy"
	BackendRequest             = Prefix + "backend_request_total"
	BackendRequestDuration     = Prefix + "backend_request_duration_seconds"
	ClientConnections          = Prefix + "client_connections_count"
	ClientConnectionsTotal     = Prefix + "client_connections_total"
	ClientRequest              = Prefix + "client_request_total"
	ClientRequestDuration      = Prefix + "client_request_duration_seconds"
)
