package transport

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/avenga/couper/config"
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
	Opts *config.HealthCheck
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

func (p Probe) String() string {
	return fmt.Sprintf("check #%d for backend %q: state: %s (%d/%d), HTTP status: %d", p.Counter, p.Name, p.State, p.Failure, p.Opts.FailureThreshold, p.Status)
}

func NewProbe(b *Backend) {
	p := &Probe{
		Log:   b.upstreamLog,
		Name:  b.name,
		Opts:  b.options.HealthCheck,
		State: StateInvalid,
	}

	go p.probe()
}

func (p *Probe) probe() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), p.Opts.Timeout)
		defer cancel()

		res, err := http.DefaultClient.Do(p.Opts.Request.WithContext(ctx))

		p.Counter++
		prevState := p.State
		p.Status = 0
		if res != nil {
			p.Status = res.StatusCode
		}

		if err != nil || !p.Opts.ExpectStatus[res.StatusCode] || !contains(res.Body, p.Opts.ExpectText) {
			if p.Failure++; p.Failure < p.Opts.FailureThreshold {
				p.State = StateDegraded
			} else {
				p.State = StateDown
			}
			p.Log.LogEntry().WithError(errors.Backend.Label(p.State.String()).With(err))
		} else {
			p.Failure = 0
			p.State = StateOk
		}

		//fmt.Println(p)
		if prevState != p.State {
			probe.BackendProbes.Store(p.Name, p.State.String())
		}

		time.Sleep(p.Opts.Interval)
	}
}

func contains(reader io.ReadCloser, text string) bool {
	if text == "" {
		return true
	}

	bytes, _ := ioutil.ReadAll(reader)
	return strings.Contains(string(bytes), text)
}
