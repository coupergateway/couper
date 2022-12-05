package definitions_test

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

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
		{"job with zero interval", fields{
			conf: &config.Job{Name: "testCase1", Interval: "0s"},
		}, "job: testCase1: interval must be a positive number", -1, 0},
		{"job with interval", fields{
			conf: &config.Job{Name: "testCase2", Interval: "200ms"},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 2, time.Millisecond * 300}, // two due to initial req
		{"job with small interval", fields{
			conf: &config.Job{Name: "testCase3", Interval: "100ms"},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("User-Agent"), "Couper") {
					getST(r).Error("expected trigger req with Couper UA")
				}
			}),
		}, "", 5, time.Millisecond * 460}, // five due to initial req
		{"job with greater interval", fields{
			conf:    &config.Job{Name: "testCase4", Interval: "1s"},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}),
		}, "", 2, time.Millisecond * 1100}, // two due to initial req
		{"job with greater origin delay than interval", fields{
			conf: &config.Job{Name: "testCase5", Interval: "1500ms"},
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
			}),
		}, "", 2, time.Second * 3}, // initial req + one (hit at 2.5s)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			h := test.New(st)
			j, err := definitions.NewJob(tt.fields.conf, tt.fields.handler, config.NewDefaultSettings())
			if tt.expErr != "" {
				if err == nil {
					h.Must(fmt.Errorf("expected an error: %v", err))
				}
				if err.Error() != tt.expErr {
					st.Errorf("\nwant err:\t%s\ngot err:\t%v", tt.expErr, err)
					return
				}
				return
			} else {
				h.Must(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx = context.WithValue(ctx, subTestKey, st)
			ctx = eval.NewDefaultContext().WithContext(ctx)

			hook.Reset()

			go j.Run(ctx, logger.WithContext(ctx))

			time.Sleep(tt.waitFor)

			logEntries := hook.AllEntries()
			cnt := 0
			for _, entry := range logEntries {
				if entry.Level != logrus.InfoLevel { // ctx cancel filter
					continue
				}
				cnt++
				msg, _ := entry.String()

				if !reflect.DeepEqual(entry.Data["name"], tt.fields.conf.Name) {
					st.Error("expected the job name in log fields")
				}

				if entry.Data["uid"].(string) == "" {
					st.Error("expected an uid log field")
				}

				defer func() {
					if st.Failed() {
						st.Log(msg)
					}
				}()
			}

			if cnt != tt.expLogs {
				st.Errorf("expected %d log entries, got: %d", tt.expLogs, len(logEntries))
			}
		})
	}
}
