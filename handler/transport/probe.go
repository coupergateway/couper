package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/avenga/couper/config/health_check"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	probe "github.com/avenga/couper/handler/transport/probe_map"
	"github.com/avenga/couper/logging"
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
	Log  *logging.UpstreamLog
	Name string
	Opts *health_check.ParsedOptions
	Req  *http.Request

	//variables reflecting status of probe
	Counter uint
	Failure uint
	State   state
	Status  int
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

func NewProbe(b *Backend) {
	p := &Probe{
		Log:  b.upstreamLog,
		Name: b.name,
		Opts: b.options.HealthCheck,
		Req:  b.options.Request,

		State: StateInvalid,
	}
	go p.probe()
}

func (p *Probe) probe() {
	for {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(p.Opts.Timeout))
		res, err := http.DefaultClient.Do(p.Req.WithContext(ctx))

		p.Counter++
		prevState := p.State
		p.State = StateInvalid
		p.Status = 0
		if err != nil {
			if p.Failure++; p.Failure <= p.Opts.FailureThreshold {
				p.State = StateDegraded
			} else {
				p.State = StateDown
			}
			p.Log.LogEntry().WithError(errors.Backend.Label(p.State.String()).With(err))
		} else {
			p.Failure = 0
			p.State = StateOk
			p.Status = res.StatusCode
		}

		//print("backend: ", p.Name, ",  state: ", p.State.Print(p.Failure, p.Opts.FailureThreshold), ",  status: ", p.Status, ",  cycle: ", p.Counter, "\n")
		if prevState != p.State {
			probe.BackendProbes.Store(p.Name, p.State.String())
		}
		cancel()
		time.Sleep(p.Opts.Interval)
	}
}
