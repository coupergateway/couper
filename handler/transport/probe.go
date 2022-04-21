package transport

import (
	"context"
	goerror "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/middleware"
	probe "github.com/avenga/couper/handler/transport/probe_map"
	"github.com/avenga/couper/logging"
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

	uidFunc middleware.UIDFunc
}

func NewProbe(log *logrus.Entry, backendName string, opts *config.HealthCheck) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: logging.NewUpstreamLog(log, http.DefaultTransport,
			false), // always false due to defaultTransport
	}

	p := &Probe{
		backendName: backendName,
		log:         log,
		opts:        opts,

		client: client,
		state:  StateInvalid,

		uidFunc: middleware.NewUIDFunc(opts.RequestUIDFormat),
	}

	go p.probe()
}

func (p *Probe) probe() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), p.opts.Timeout)
		ctx = context.WithValue(ctx, request.RoundTripName, "health-check")
		uid := p.uidFunc()
		ctx = context.WithValue(ctx, request.UID, uid)

		res, err := p.client.Do(p.opts.Request.Clone(ctx))
		cancel()

		p.counter++
		prevState := p.state
		p.status = 0
		if res != nil {
			p.status = res.StatusCode
		}

		var errorMessage string
		if err != nil || !p.opts.ExpectedStatus[res.StatusCode] || !contains(res.Body, p.opts.ExpectedText) {
			if p.failure++; p.failure < p.opts.FailureThreshold {
				p.state = StateFailing
			} else {
				p.state = StateDown
			}
			if err == nil {
				if !p.opts.ExpectedStatus[res.StatusCode] {
					errorMessage = "unexpected statusCode: " + strconv.Itoa(p.status)
				} else {
					errorMessage = "unexpected text"
				}
			} else {
				unwrapped := goerror.Unwrap(err)
				if unwrapped != nil {
					err = unwrapped
				}
				if gerr, ok := err.(errors.GoError); ok {
					errorMessage = gerr.LogError()
				} else {
					errorMessage = err.Error()
				}
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

			log := p.log.WithField("uid", uid)
			switch p.state {
			case StateOk:
				log.Info(message)
			case StateFailing:
				log.Warn(message)
			case StateDown:
				log.WithError(fmt.Errorf(errorMessage + ": " + message)).Error()
			}
		}

		time.Sleep(p.opts.Interval)
	}
}

func (p Probe) String() string {
	return fmt.Sprintf("check #%d for backend %q: state: %s (%d/%d), HTTP status: %d", p.counter, p.backendName, p.state, p.failure, p.opts.FailureThreshold, p.status)
}

func contains(reader io.ReadCloser, text string) bool {
	defer reader.Close() // free resp body related connection

	if text == "" {
		return true
	}

	bytes, _ := io.ReadAll(reader)
	return strings.Contains(string(bytes), text)
}
