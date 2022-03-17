package transport

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
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

type Probe struct {
	//configurable settings
	log  *logrus.Entry
	Name string
	Opts *config.HealthCheck

	//variables reflecting status of probe
	Counter uint
	Failure uint
	State   state
	Status  int
	client  *http.Client
}

func (s state) String() string {
	return healthStateLabels[s]
}

func (p Probe) String() string {
	return fmt.Sprintf("check #%d for backend %q: state: %s (%d/%d), HTTP status: %d", p.Counter, p.Name, p.State, p.Failure, p.Opts.FailureThreshold, p.Status)
}

func NewProbe(log *logrus.Entry, backendName string, opts *config.HealthCheck) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p := &Probe{
		log:    log,
		Name:   backendName,
		Opts:   opts,
		State:  StateInvalid,
		client: client,
	}

	go p.probe()
}

func (p *Probe) probe() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), p.Opts.Timeout)
		defer cancel()

		res, err := p.client.Do(p.Opts.Request.WithContext(ctx))

		p.Counter++
		prevState := p.State
		p.Status = 0
		if res != nil {
			p.Status = res.StatusCode
		}

		var errorMessage string
		if err != nil || !p.Opts.ExpectStatus[res.StatusCode] || !contains(res.Body, p.Opts.ExpectText) {
			if p.Failure++; p.Failure < p.Opts.FailureThreshold {
				p.State = StateFailing
			} else {
				p.State = StateDown
			}
			if err == nil {
				errorMessage = "Unexpected status or text"
				err = fmt.Errorf(errorMessage)
			} else {
				errorMessage = err.Error()
			}
		} else {
			p.Failure = 0
			p.State = StateOk
			errorMessage = ""
		}

		if prevState != p.State {
			newState := p.State.String()
			probe.BackendProbes.Store(p.Name, probe.HealthInfo{
				State:   newState,
				Error:   errorMessage,
				Healthy: p.State != StateDown,
			})

			message := fmt.Sprintf("new health state: %s", newState)

			switch p.State {
			case StateOk:
				p.log.Info(message)
			case StateFailing:
				p.log.Warn(message)
			case StateDown:
				p.log.WithError(errors.Backend.
					Label(p.Name).
					Message(message).
					With(err)).Error()
			}
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
