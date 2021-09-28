package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"

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
	name             string
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

func (b *Backend) getConfig() (*Config, error) {
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	origin := b.transportConf.Scheme + "://" + b.transportConf.Origin
	req = req.WithContext(context.WithValue(context.Background(), request.URLAttribute, origin))
	return b.evalTransport(req)
}

func (b *Backend) NewProbe() {
	p := &Probe{
		name:             b.name,
		failureThreshold: b.transportConf.HealthCheck.FailureThreshold,
		time:             b.transportConf.HealthCheck.Period,
		timeOut:          b.transportConf.HealthCheck.Timeout,

		counter: 0,
		failure: 0,
	}
	probe.SetBackendProbe(p.name, StateInvalid.String())
	c, err := b.getConfig()
	if err != nil {
		b.upstreamLog.LogEntry().WithError(err).Error()
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
		probe.SetBackendProbe(p.name, p.state.String())
		cancel()
		time.Sleep(p.time)
	}
}
