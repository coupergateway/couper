package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type probe struct {
	//configurable settings
	backend          *Backend
	failureThreshold int
	time             time.Duration
	timeOut          time.Duration

	//variables reflecting status of probe
	counter int
	failure int
	state   string
	status  int
}

func newProbe(time, timeOut time.Duration, failureThreshold int, backend *Backend) {
	p := &probe{
		backend:          backend,
		failureThreshold: failureThreshold,
		time:             time,
		timeOut:          timeOut,

		counter: 1,
		failure: 0,
	}
	go p.probe()
}

func (p *probe) probe() {
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	c, err := p.backend.evalTransport(req)
	if err != nil {
		p.backend.upstreamLog.LogEntry().WithError(err).Error()
		return
	}
	req, _ = http.NewRequest(http.MethodGet, c.Scheme+"://"+c.Origin, nil)

	for {
		time.Sleep(p.time)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(p.timeOut))
		res, err := http.DefaultClient.Do(req.WithContext(ctx))

		p.state = "OK"
		p.status = 0
		if err != nil {
			if p.failure++; p.failure <= p.failureThreshold {
				p.state = "DEGRADED " + strconv.Itoa(p.failure) + "/" + strconv.Itoa(p.failureThreshold)
			} else {
				p.state = "DOWN " + strconv.Itoa(p.failure) + "/" + strconv.Itoa(p.failureThreshold)
			}
		} else {
			p.failure = 0
			p.status = res.StatusCode
		}

		print("healthcheck ", p.counter, ", state ", p.state, ", status code ", p.status, "\n")
		cancel()
		p.counter++
	}
}
