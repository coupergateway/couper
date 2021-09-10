package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

const (
	stateInvalid = iota
	stateOk
	stateDegraded
	stateDown
)

type state int

type probe struct {
	//configurable settings
	backend          *Backend
	failureThreshold int
	time             time.Duration
	timeOut          time.Duration

	//variables reflecting status of probe
	counter int
	failure int
	state   state
	status  int
}

func (state *state) toString(f int, ft int) string {
	switch *state {
	case stateOk:
		return "OK"
	case stateDegraded:
		return "DEGRADED " + strconv.Itoa(f) + "/" + strconv.Itoa(ft)
	case stateDown:
		return "DOWN " + strconv.Itoa(f) + "/" + strconv.Itoa(ft)
	default:
		return "INVALID"
	}
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

		p.state = stateInvalid
		p.status = 0
		if err != nil {
			if p.failure++; p.failure <= p.failureThreshold {
				p.state = stateDegraded
			} else {
				p.state = stateDown
			}
		} else {
			p.failure = 0
			p.state = stateOk
			p.status = res.StatusCode
		}

		print("healthcheck ", p.counter, ", state ", p.state.toString(p.failure, p.failureThreshold), ", status code ", p.status, "\n")
		cancel()
		p.counter++
	}
}
