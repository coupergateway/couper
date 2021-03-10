package runtime

import (
	"time"
)

type HTTPTimings struct {
	IdleTimeout       time.Duration `env:"timing_idle_timeout"`
	ReadHeaderTimeout time.Duration `env:"timing_read_header_timeout"`
	// ShutdownDelay determines the time between marking the http server
	// as unhealthy and calling the final shutdown method which denies accepting new requests.
	ShutdownDelay time.Duration `env:"timing_shutdown_delay"`
	// ShutdownTimeout is the context duration for shutting down the http server. Running requests
	// gets answered and those which exceeded this timeout getting lost. In combination with
	// ShutdownDelay the load-balancer should have picked another instance already.
	ShutdownTimeout time.Duration `env:"timing_shutdown_timeout"`
}

var DefaultTimings = HTTPTimings{
	IdleTimeout:       time.Second * 60,
	ReadHeaderTimeout: time.Second * 10,
}
