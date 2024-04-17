package definitions

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/variables"
	"github.com/coupergateway/couper/handler/middleware"
	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/server/writer"
	"github.com/coupergateway/couper/utils"
)

type Job struct {
	conf     *config.Job
	handler  http.Handler
	interval time.Duration
	settings *config.Settings
}

type Jobs []*Job

func (j Jobs) Run(ctx context.Context, log *logrus.Entry) {
	if len(j) == 0 {
		return
	}

	logEntry := log.WithContext(ctx)
	logEntry.Data["type"] = "couper_job"

	for _, job := range j {
		go job.Run(ctx, logEntry)
	}
}

func NewJob(j *config.Job, h http.Handler, settings *config.Settings) *Job {
	return &Job{
		conf:     j,
		handler:  h,
		interval: j.IntervalDuration,
		settings: settings,
	}
}

func (j *Job) Run(ctx context.Context, logEntry *logrus.Entry) {
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	req.Header.Set("User-Agent", "Couper / "+utils.VersionName+" job-"+j.conf.Name)

	uidFn := middleware.NewUIDFunc(j.settings.RequestIDBackendHeader)

	t := time.NewTicker(time.Millisecond * 50)
	defer t.Stop()

	firstRun := true

	clh := middleware.NewCustomLogsHandler([]hcl.Body{j.conf.Remain}, j.handler, j.conf.Name)

	for {
		select {
		case <-ctx.Done():
			logEntry.WithFields(logrus.Fields{
				"name": j.conf.Name,
			}).Errorf("stopping: %v", ctx.Err())
			return
		case <-t.C:
			uid := uidFn()

			outReq := req.Clone(context.WithValue(ctx, request.UID, uid))

			evalCtx := eval.ContextFromRequest(outReq).WithClientRequest(outReq) // setup syncMap, upstream custom logs
			delete(evalCtx.HCLContext().Variables, variables.ClientRequest)      // this is the noop req from above, not helpful

			outCtx := context.WithValue(evalCtx, request.LogEntry, logEntry)
			outCtx = context.WithValue(outCtx, request.LogCustomAccess, []hcl.Body{j.conf.Remain}) // local custom logs
			outReq = outReq.WithContext(outCtx)

			n := time.Now()
			log := logEntry.
				WithFields(logrus.Fields{
					"name": j.conf.Name,
					"timings": logging.Fields{
						"total":    logging.RoundMS(time.Since(n)),
						"interval": logging.RoundMS(j.interval),
					},
					"uid": uid,
				}).WithContext(outCtx).
				WithTime(n)

			go run(outReq, clh, log)

			if firstRun {
				t.Reset(j.interval)
				firstRun = false
			}
		}
	}
}

func run(req *http.Request, h http.Handler, log *logrus.Entry) {
	w := writer.NewResponseWriter(&noopResponseWriter{}, "")
	h.ServeHTTP(w, req)

	if w.StatusCode() == 0 || w.StatusCode() > 499 {
		if ctxErr, ok := req.Context().Value(request.Error).(*errors.Error); ok {
			if len(ctxErr.Kinds()) > 0 {
				log = log.WithFields(logrus.Fields{"error_type": ctxErr.Kinds()[0]})
			}
			log.Error(ctxErr.Error())
			return
		}
		log.Error()
		return
	}
	log.Info()
}

var _ http.ResponseWriter = &noopResponseWriter{}

type noopResponseWriter struct {
	header http.Header
}

func (n *noopResponseWriter) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}
	return n.header
}

func (n *noopResponseWriter) Write(bytes []byte) (int, error) {
	return len(bytes), nil
}

func (n *noopResponseWriter) WriteHeader(_ int) {}
