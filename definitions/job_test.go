package definitions_test

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/definitions"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/test"
)

func TestJob_Run(t *testing.T) {
	type fields struct {
		conf    *config.Job
		handler http.Handler
	}

	logger, hook := test.NewLogger()

	const subTestKey = "subTest"
	getST := func(r *http.Request) *testing.T {
		return eval.ContextFromRequest(r).Value(subTestKey).(*testing.T)
	}

	tests := []struct {
		name    string
		fields  fields
		expErr  string
		expLogs int
		waitFor time.Duration
	}{
		{"job with interval", fields{
			conf: &config.Job{Name: "testCase1", IntervalDuration: time.Millisecond * 200},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 2, time.Millisecond * 300}, // two due to initial req
		{"job with small interval", fields{
			conf: &config.Job{Name: "testCase2", IntervalDuration: time.Millisecond * 100},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 5, time.Millisecond * 410}, // five due to initial req
		{"job with greater interval", fields{
			conf:    &config.Job{Name: "testCase3", IntervalDuration: time.Second},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}),
		}, "", 2, time.Millisecond * 1100}, // two due to initial req
		{"job with greater origin delay than interval", fields{
			conf: &config.Job{Name: "testCase4", IntervalDuration: time.Millisecond * 1500},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
			}),
		}, "", 2, time.Second * 3}, // initial req + one (hit at 2.5s)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			j := definitions.NewJob(tt.fields.conf, tt.fields.handler, config.NewDefaultSettings())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx = context.WithValue(ctx, subTestKey, st)
			ctx = eval.NewDefaultContext().WithContext(ctx)

			hook.Reset()

			go j.Run(ctx, logger.WithContext(ctx))

			// 50ms are the initial ticker delay
			time.Sleep(tt.waitFor + (time.Millisecond * 50))

			logEntries := hook.AllEntries()
			for _, entry := range logEntries {
				msg, _ := entry.String()
				if strings.Contains(msg, "context canceled") {
					continue
				}

				if !reflect.DeepEqual(entry.Data["name"], tt.fields.conf.Name) {
					st.Error("expected the job name in log fields")
				}

				if uid, _ := entry.Data["uid"].(string); uid == "" {
					st.Error("expected an uid log field")
				}

				defer func() {
					if st.Failed() {
						st.Log(msg)
					}
				}()
			}

			if len(logEntries) != tt.expLogs {
				st.Errorf("expected %d log entries, got: %d", tt.expLogs, len(logEntries))
			}
		})
	}
}
