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

type HealthInfo struct {
	Error   string
	Healthy bool
	Origin  string
	State   string
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

	listener ProbeStateChange

	uidFunc middleware.UIDFunc
}

type ProbeStateChange interface {
	OnProbeChange(info *HealthInfo)
}

func NewProbe(log *logrus.Entry, tc *Config, opts *config.HealthCheck, listener ProbeStateChange) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: logging.NewUpstreamLog(log,
			NewTransport(tc.
				WithTarget(opts.Request.URL.Scheme, opts.Request.URL.Host, opts.Request.URL.Host, ""),
				log),
			tc.NoProxyFromEnv),
	}

	p := &Probe{
		backendName: tc.BackendName,
		log:         log.WithField("url", opts.Request.URL.String()),
		opts:        opts,

		client: client,
		state:  StateInvalid,

		listener: listener,

		uidFunc: middleware.NewUIDFunc(opts.RequestUIDFormat),
	}

	// do not start go-routine on config check (-watch)
	if _, exist := opts.Context.Value(request.ConfigDryRun).(bool); exist {
		return
	}
	go p.probe(opts.Context)
}

func (p *Probe) probe(c context.Context) {
	for {
		select {
		case <-c.Done():
			p.log.Warn("shutdown health probe")
			return
		default:
		}
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
					errorMessage = "unexpected status code: " + strconv.Itoa(p.status)
				} else {
					errorMessage = "unexpected text"
				}
			} else {
				unwrapped := goerror.Unwrap(err)
				if unwrapped != nil {
					err = unwrapped
				}

				if gerr, ok := err.(errors.GoError); ok {
					// Upstream log wraps a possible transport deadline into a backend error
					if gerr.Unwrap() == context.DeadlineExceeded {
						errorMessage = fmt.Sprintf("backend error: connecting to %s '%s' failed: i/o timeout",
							p.backendName, p.opts.Request.URL.Hostname())
					} else {
						errorMessage = gerr.LogError()
					}
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
			info := &HealthInfo{
				Error:   errorMessage,
				Healthy: p.state != StateDown,
				Origin:  p.opts.Request.URL.Host,
				State:   newState,
			}

			if p.listener != nil {
				p.listener.OnProbeChange(info)
			}

			message := fmt.Sprintf("new health state: %s", newState)

			log := p.log.WithField("uid", uid)
			switch p.state {
			case StateOk:
				log.Info(message)
			case StateFailing:
				log.Warn(message)
			case StateDown:
				log.WithError(errors.BackendUnhealthy.Message(errorMessage + ": " + message)).Error()
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
