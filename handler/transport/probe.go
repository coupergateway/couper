package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"

	probe "github.com/avenga/couper/handler/transport/probe_map"
)

const (
	stateInvalid = iota
	stateOk
	stateDegraded
	stateDown
)

type state int

type Probe struct {
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

func (state *state) String() string {
	switch *state {
	case stateOk:
		return "OK"
	case stateDegraded:
		return "DEGRADED"
	case stateDown:
		return "DOWN"
	default:
		return "INVALID"
	}
}

func (state *state) Print(f int, ft int) string {
	if f != 0 {
		return state.String() + " " + strconv.Itoa(f) + "/" + strconv.Itoa(ft)
	}
	return state.String()
}

func NewProbe(time, timeOut time.Duration, failureThreshold int, backend *Backend) {
	p := &Probe{
		backend:          backend,
		failureThreshold: failureThreshold,
		time:             time,
		timeOut:          timeOut,

		counter: 0,
		failure: 0,
	}
	go p.probe()
	//go p.check()
}

/*func (p *probe) check() {
	for {
		time.Sleep(p.time)
		print("name: ", p.backend.name, ", state: ", backend_probes.name(p.backend.name), "\n")
	}
}*/

func (p *Probe) probe() {
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

		p.counter++
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

		print("healthcheck ", p.counter, ", state ", p.state.Print(p.failure, p.failureThreshold), ", status code ", p.status, "\n")
		probe.SetBackendProbe(p.backend.name, p.state.String())
		cancel()
	}
}
