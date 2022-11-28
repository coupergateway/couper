package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/server/writer"
)

type Job struct {
	ctx      context.Context
	handler  *handler.Endpoint
	job      *config.Job
	settings *config.Settings
}

var jobs = make(map[string]*Job)

func Start(ctx context.Context, log *logrus.Entry) {
	if len(jobs) == 0 {
		return
	}

	for name, job := range jobs {
		go func(job *Job, name string) {
			// TODO: Does not work...
			logEntry := log.WithField("type", "job")

			defer func() {
				if rc := recover(); rc != nil {
					logEntry.WithField("panic", rc).Error()
				}
			}()

			req := &http.Request{
				Header: http.Header{},
				URL:    &url.URL{},
			}
			req.Header.Set("User-Agent", name)

			outCtx := context.WithValue(ctx, request.LogEntry, logEntry)
			req = req.WithContext(outCtx)

			uidHandler := middleware.NewUIDHandler(job.GetSettings(), "")(job.GetHandler())

			// Execute initial, do not wait for execution
			go uidHandler.ServeHTTP(writer.NewResponseWriter(httptest.NewRecorder(), ""), req)

			interval, _ := time.ParseDuration(job.GetJob().Interval)

			for {
				select {
				case <-job.GetCtx().Done():
					return
				case <-time.After(interval):
					uidHandler = middleware.NewUIDHandler(job.GetSettings(), "")(job.GetHandler())

					// Do not wait for execution
					go uidHandler.ServeHTTP(writer.NewResponseWriter(httptest.NewRecorder(), ""), req)
				}
			}
		}(job, name)
	}
}

func AddJob(ctx context.Context, j *config.Job, h *handler.Endpoint, settings *config.Settings) {
	jobs[j.Name] = &Job{
		ctx:      ctx,
		handler:  h,
		job:      j,
		settings: settings,
	}
}

func (j *Job) GetCtx() context.Context {
	return j.ctx
}

func (j *Job) GetHandler() *handler.Endpoint {
	return j.handler
}

func (j *Job) GetJob() *config.Job {
	return j.job
}

func (j *Job) GetSettings() *config.Settings {
	return j.settings
}
