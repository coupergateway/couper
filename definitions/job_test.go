package definitions_test

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/definitions"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/test"
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
		waitFor time.Duration // only used when expLogs == 0
	}{
		{"job with interval", fields{
			conf: &config.Job{Name: "testCase1", IntervalDuration: time.Millisecond * 200},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 2, 0}, // two due to initial req
		{"job with small interval", fields{
			conf: &config.Job{Name: "testCase2", IntervalDuration: time.Millisecond * 100},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 5, 0}, // five due to initial req
		{"job with greater interval", fields{
			conf:    &config.Job{Name: "testCase3", IntervalDuration: time.Second},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}),
		}, "", 2, 0}, // two due to initial req
		{"job with greater origin delay than interval", fields{
			conf: &config.Job{Name: "testCase4", IntervalDuration: time.Millisecond * 1500},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
			}),
		}, "", 2, 0}, // initial req + one (hit at 2.5s)
		{"job with startup_delay", fields{
			conf: &config.Job{Name: "testCase5", IntervalDuration: time.Millisecond * 100, StartupDelayDuration: time.Millisecond * 300},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 0, time.Millisecond * 200}, // no execution within 200ms due to startup_delay of 300ms
		{"job with startup_delay executes after delay", fields{
			conf: &config.Job{Name: "testCase6", IntervalDuration: time.Millisecond * 100, StartupDelayDuration: time.Millisecond * 200},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 2, 0}, // first at ~200ms (startup_delay), second at ~300ms (interval)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			var counter atomic.Int32
			reached := make(chan struct{}, 1)
			expected := int32(tt.expLogs)

			wrappedHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				tt.fields.handler.ServeHTTP(rw, r)
				if expected > 0 && counter.Add(1) == expected {
					select {
					case reached <- struct{}{}:
					default:
					}
				}
			})

			j := definitions.NewJob(tt.fields.conf, wrappedHandler, config.NewDefaultSettings())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx = context.WithValue(ctx, subTestKey, st)
			ctx = eval.NewDefaultContext().WithContext(ctx)

			hook.Reset()

			go j.Run(ctx, logger.WithContext(ctx))

			if tt.expLogs > 0 {
				select {
				case <-reached:
					// Expected execution count reached
				case <-time.After(5 * time.Second):
					st.Fatalf("timed out waiting for %d executions, got %d", tt.expLogs, counter.Load())
				}
			} else {
				// For zero-expectation, wait and verify nothing ran
				time.Sleep(tt.waitFor + (time.Millisecond * 50))
				if cnt := counter.Load(); cnt != 0 {
					st.Errorf("expected 0 executions, got %d", cnt)
				}
			}

			// Let log writes complete after last handler execution
			time.Sleep(50 * time.Millisecond)

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
		})
	}
}
