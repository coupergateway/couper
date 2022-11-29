package definitions

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/server/writer"
	"github.com/avenga/couper/utils"
)

type Job struct {
	conf     *config.Job
	handler  *handler.Endpoint
	interval time.Duration
	settings *config.Settings
}

type Jobs []*Job

func (j Jobs) Run(ctx context.Context, log *logrus.Entry) {
	if len(j) == 0 {
		return
	}

	logEntry := log.WithContext(ctx)
	logEntry.Data["type"] = "job"

	for _, job := range j {
		go job.Run(ctx, logEntry)
	}
}

func NewJob(j *config.Job, h *handler.Endpoint, settings *config.Settings) (*Job, error) {
	interval, err := time.ParseDuration(j.Interval)
	if err != nil {
		return nil, err
	}
	return &Job{
		conf:     j,
		handler:  h,
		interval: interval,
		settings: settings,
	}, nil
}

func (j *Job) Run(ctx context.Context, logEntry *logrus.Entry) {
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	req.Header.Set("User-Agent", "Couper / "+utils.VersionName+" conf-"+j.conf.Name)

	t := time.NewTicker(time.Millisecond)
	defer t.Stop()

	firstRun := true

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			evalCtx := ctx.(*eval.Context)
			evalCtx = evalCtx.WithClientRequest(req) // setup syncMap, custom logs
			outCtx := context.WithValue(evalCtx, request.LogEntry, logEntry)
			outReq := req.Clone(outCtx)

			w := writer.NewResponseWriter(&noopResponseWriter{}, "")
			j.handler.ServeHTTP(w, outReq)

			if firstRun {
				firstRun = false
				t.Reset(j.interval)
			}
		}
	}
}

var _ http.ResponseWriter = &noopResponseWriter{}

type noopResponseWriter struct {
	header http.Header
}

func (n noopResponseWriter) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}
	return n.header
}

func (n noopResponseWriter) Write(bytes []byte) (int, error) {
	return len(bytes), nil
}

func (n noopResponseWriter) WriteHeader(_ int) {}
