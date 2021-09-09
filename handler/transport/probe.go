package transport

import (
	"net/http"
	"strconv"
	"time"
)

type probeOptions struct {
	time             time.Duration
	failureThreshold int
}

func (c *Config) Probe(b *Backend) {
	probeOpts := &probeOptions{
		time:             time.Second,
		failureThreshold: 5,
	}
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	c, _ = b.evalTransport(req)
	req, _ = http.NewRequest(http.MethodGet, c.Scheme+"://"+c.Origin, nil)

	for counter, failure := 1, 0; true; counter++ {
		time.Sleep(probeOpts.time)
		_, err := http.DefaultClient.Do(req)

		state := "OK"
		if err == nil {
			failure = 0
		} else if failure++; failure <= probeOpts.failureThreshold {
			state = "DEGRADED " + strconv.Itoa(failure) + "/" + strconv.Itoa(probeOpts.failureThreshold)
		} else {
			state = "DOWN " + strconv.Itoa(failure) + "/" + strconv.Itoa(probeOpts.failureThreshold)
		}

		println("healthcheck", counter, state)
	}
}
