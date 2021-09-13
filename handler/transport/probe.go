package transport

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	stateInvalid = iota
	stateOk
	stateDegraded
	stateDown
)

type state int

type ProbeMap struct {
	nm map[string]string
	mu sync.Mutex
}

var backend_probes = &ProbeMap{
	nm: make(map[string]string),
}

func (pm *ProbeMap) Set(p *probe) {
	pm.mu.Lock()
	pm.nm[p.origin] = p.state.String()
	pm.mu.Unlock()
}

func (pm *ProbeMap) name(name string) string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.nm[name]
}

func (pm *ProbeMap) Del(n string) {
	pm.mu.Lock()
	delete(pm.nm, n)
	pm.mu.Unlock()
}

type probe struct {
	//configurable settings
	backend          *Backend
	failureThreshold int
	origin           string
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
	p := &probe{
		backend:          backend,
		failureThreshold: failureThreshold,
		time:             time,
		timeOut:          timeOut,

		counter: 0,
		failure: 0,
	}
	go p.probe()
	go p.check()
}

func (p *probe) check() {
	for {
		time.Sleep(p.time)
		print("name: ", p.origin, ", state: ", backend_probes.name(p.origin), "\n")
	}
}

func (p *probe) probe() {
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	c, err := p.backend.evalTransport(req)
	if err != nil {
		p.backend.upstreamLog.LogEntry().WithError(err).Error()
		return
	}
	p.origin = c.Scheme + "://" + c.Origin
	req, _ = http.NewRequest(http.MethodGet, p.origin, nil)

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
		backend_probes.Set(p)
		cancel()
	}
}
