package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	probe "github.com/avenga/couper/handler/transport/probe_map"
)

const (
	StateInvalid state = iota
	StateOk
	StateFailing
	StateDown
)

var healthStateLabels = []string{
	"invalid",
	"healthy",
	"failing",
	"unhealthy",
}

var _ context.Context = &eval.Context{}

type state int

func (s state) String() string {
	return healthStateLabels[s]
}

type Probe struct {
	//configurable settings
	backendName string
	log         *logrus.Entry
	opts        *config.HealthCheck

	//variables reflecting status of probe
	client  *http.Client
	counter uint
	failure uint
	state   state
	status  int
}

func NewProbe(log *logrus.Entry, backendName string, opts *config.HealthCheck) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p := &Probe{
		backendName: backendName,
		log:         log,
		opts:        opts,

		client: client,
		state:  StateInvalid,
	}

	go p.probe()
}

func (p *Probe) probe() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), p.opts.Timeout)
		defer cancel()

		res, err := p.client.Do(p.opts.Request.WithContext(ctx))

		p.counter++
		prevState := p.state
		p.status = 0
		if res != nil {
			p.status = res.StatusCode
		}

		var errorMessage string
		if err != nil || !p.opts.ExpectStatus[res.StatusCode] || !contains(res.Body, p.opts.ExpectText) {
			if p.failure++; p.failure < p.opts.FailureThreshold {
				p.state = StateFailing
			} else {
				p.state = StateDown
			}
			if err == nil {
				errorMessage = "Unexpected status or text"
				err = fmt.Errorf(errorMessage)
			} else {
				errorMessage = err.Error()
			}
		} else {
			p.failure = 0
			p.state = StateOk
			errorMessage = ""
		}

		if prevState != p.state {
			newState := p.state.String()
			probe.BackendProbes.Store(p.backendName, probe.HealthInfo{
				State:   newState,
				Error:   errorMessage,
				Healthy: p.state != StateDown,
			})

			message := fmt.Sprintf("new health state: %s", newState)

			switch p.state {
			case StateOk:
				p.log.Info(message)
			case StateFailing:
				p.log.Warn(message)
			case StateDown:
				p.log.WithError(errors.Backend.
					Label(p.backendName).
					Message(message).
					With(err)).Error()
			}
		}

		time.Sleep(p.opts.Interval)
	}
}

func (p Probe) String() string {
	return fmt.Sprintf("check #%d for backend %q: state: %s (%d/%d), HTTP status: %d", p.counter, p.backendName, p.state, p.failure, p.opts.FailureThreshold, p.status)
}

func contains(reader io.ReadCloser, text string) bool {
	if text == "" {
		return true
	}

	bytes, _ := io.ReadAll(reader)
	return strings.Contains(string(bytes), text)
}
