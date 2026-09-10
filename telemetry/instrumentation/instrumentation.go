package instrumentation

const (
	Name   = "github.com/coupergateway/couper/telemetry"
	Prefix = "couper_"

	BackendInstrumentationName       = "couper/backend"
	AccessControlInstrumentationName = "couper/access_control"

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

	AccessControlTotal           = Prefix + "access_control_total"
	AccessControlDuration        = Prefix + "access_control_duration_seconds"
	AccessControlRateLimited     = Prefix + "access_control_rate_limited_total"
	AccessControlRateLimiterKeys = Prefix + "access_control_rate_limiter_active_keys"
)

// DefaultDurationSecondsBoundaries are the boundaries the OpenTelemetry semantic conventions
// specify for HTTP request duration. The SDK's own defaults assume milliseconds.
var DefaultDurationSecondsBoundaries = []float64{
	0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
}

// ConnectionLifetimeSecondsBoundaries cover the lifetime of a backend connection. Couper
// leaves the transport's IdleConnTimeout unset, so a pooled keep-alive connection lives
// until the origin closes it — minutes or hours rather than the seconds a request takes.
// DefaultDurationSecondsBoundaries would collapse nearly every observation into +Inf.
var ConnectionLifetimeSecondsBoundaries = []float64{
	0.1, 0.5, 1, 5, 10, 30, 60, 300, 900, 1800, 3600, 7200,
}
