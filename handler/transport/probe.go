package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/avenga/couper/eval/content"

	"github.com/avenga/couper/config/request"

	"github.com/avenga/couper/eval"
	probe "github.com/avenga/couper/handler/transport/probe_map"
)

const (
	StateInvalid state = iota
	StateOk
	StateDegraded
	StateDown
)

var _ context.Context = &eval.Context{}

type state int

type Probe struct {
	//configurable settings
	backend          *Backend
	config           *Config
	failureThreshold int
	time             time.Duration
	timeOut          time.Duration

	//variables reflecting status of probe
	counter int
	failure int
	state   state
	status  int
}

func (s state) String() string {
	switch s {
	case StateOk:
		return "OK"
	case StateDegraded:
		return "DEGRADED"
	case StateDown:
		return "DOWN"
	default:
		return "INVALID"
	}
}

func (state state) Print(f int, ft int) string {
	if f != 0 {
		return state.String() + " " + strconv.Itoa(f) + "/" + strconv.Itoa(ft)
	}
	return state.String()
}

func (p *Probe) getConfig() (*Config, error) {
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	origin, _ := content.GetContextAttribute(p.backend.confContext, p.backend.context, "origin")
	req = req.WithContext(context.WithValue(p.backend.confContext, request.URLAttribute, origin))
	return p.backend.evalTransport(req)
}

func NewProbe(backend *Backend) {
	p := &Probe{
		backend:          backend,
		failureThreshold: backend.transportConf.HealthCheck.FailureThreshold,
		time:             backend.transportConf.HealthCheck.Period,
		timeOut:          backend.transportConf.HealthCheck.Timeout,

		counter: 0,
		failure: 0,
	}
	probe.SetBackendProbe(p.backend.name, StateInvalid.String())
	c, err := p.getConfig()
	if err != nil {
		p.backend.upstreamLog.LogEntry().WithError(err).Error()
		return
	}
	p.config = c
	go p.probe()
}

func (p *Probe) probe() {
	req, _ := http.NewRequest(http.MethodGet, p.config.Scheme+"://"+p.config.Origin, nil)

	for {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(p.timeOut))
		res, err := http.DefaultClient.Do(req.WithContext(ctx))

		p.counter++
		p.state = StateInvalid
		p.status = 0
		if err != nil {
			if p.failure++; p.failure <= p.failureThreshold {
				p.state = StateDegraded
			} else {
				p.state = StateDown
			}
		} else {
			p.failure = 0
			p.state = StateOk
			p.status = res.StatusCode
		}

		//print("backend: ", p.backend.name, ",  state: ", p.state.Print(p.failure, p.failureThreshold), ",  status: ", p.status, ",  cycle: ", p.counter, "\n")
		probe.SetBackendProbe(p.backend.name, p.state.String())
		cancel()
		time.Sleep(p.time)
	}
}
