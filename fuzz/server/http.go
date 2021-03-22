//+build gofuzz

package server

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config/configload"
	couperruntime "github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server"
)

var client *http.Client
var logs io.Reader

func init() {
	upstream := test.NewBackend()

	configFileContent := fmt.Sprintf(`server "fuzz" {
		endpoint "/**" {
			add_request_headers = {
				x-fuzz = req.headers.x-data
			}

			add_query_params = {
				x-quzz = req.headers.x-data
			}

			request "sidekick" {
				url = "http://%s/anything/"
				body
			}
			
			# default
			proxy {
				path = "/anything"
				url = "http://%s"
			}

			add_response_headers = {
				y-fuzz = req.headers.x-data
			}
		}
}`, upstream.Addr(), upstream.Addr())

	configFile, err := configload.LoadBytes([]byte(configFileContent), "fuzz_http.hcl")
	if err != nil {
		panic(err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	logs = r
	logger := logrus.New()
	logger.Out = w
	log := logger.WithField("fuzz", "server/http")

	cmdCtx := command.ContextWithSignal(context.Background())
	config, err := couperruntime.NewServerConfiguration(configFile, log, cache.New(log, cmdCtx.Done()))
	if err != nil {
		panic(err)
	}

	servers, fn := server.NewServerList(cmdCtx, configFile.Context, log, configFile.Settings, &couperruntime.DefaultTimings, config)
	for _, s := range servers {
		s.Listen()
	}
	go fn()

	client = &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second,
			MaxConnsPerHost: 0,
		},
	}
}

func Fuzz(data []byte) int {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	if err != nil {
		panic(err)
	}

	req.URL.Path += string(data)

	req.Header.Set("X-Data", string(data))
	res, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "net/http: invalid") {
			panic(err)
			return 0 // useless input, invalid http
		}
		return 1 // useful input
	}

	if res.StatusCode > 499 { // useful fuzz input
		return 1
	}

	logData, err := ioutil.ReadAll(logs)
	if err != nil {
		panic(err)
	}

	if strings.Contains(string(logData), "panic:") {
		panic("couper panic")
		return 1
	}

	if runtime.NumGoroutine() > 100 {
		panic("goroutine leak")
		return 1
	}

	return 0
}
