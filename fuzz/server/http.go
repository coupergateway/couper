package server

import (
	"context"
	"fmt"
	"io"
	"net"
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

	configFileContent := fmt.Sprintf(`
server "fuzz" {
	endpoint "/**" {
		add_request_headers = {
			x-fuzz = request.headers.x-data
		}

		add_query_params = {
			x-quzz = request.headers.x-data
		}

		request "sidekick" {
			url = "http://%s/anything/"
			body = request.headers.x-data
		}
		
		# default
		proxy {
			path = "/anything"
			url = "http://%s"
		}

		add_response_headers = {
			y-fuzz = request.headers.x-data
			x-sidekick = backend_responses.sidekick.json_body
		}
	}
}

settings {
  default_port = 0
  no_proxy_from_env = true
}
`, upstream.Addr(), upstream.Addr())

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
		panic("init error: " + err.Error())
	}

	servers, _, err := server.NewServers(cmdCtx, configFile.Context, log, configFile.Settings, &couperruntime.DefaultTimings, config)
	if err != nil {
		panic("init error: " + err.Error())
	}

	var addr string
	for _, s := range servers {
		if err = s.Listen(); err != nil {
			panic("init error: " + err.Error())
		} else {
			addr = s.Addr()
			break // support just one server
		}
	}

	d := &net.Dialer{Timeout: time.Second}
	client = &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second,
			MaxConnsPerHost: 0,
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return d.DialContext(ctx, "tcp4", addr)
			},
		},
	}
}

func Fuzz(data []byte) int {
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	req.URL.Path += string(data)

	req.Header.Set("X-Data", string(data))

	values := req.URL.Query()
	values.Add("X-Data", string(data))
	req.URL.RawQuery = values.Encode()

	res, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "net/http: invalid") {
			return 0 // useless input, invalid http
		}
		panic(err)
		return 1 // useful input
	}

	logData, err := io.ReadAll(logs)
	if err != nil {
		panic("reading log-data:" + err.Error())
	}

	if res.Header.Get("y-fuzz") != string(data) {
		panic("request / response data are not equal:\n" + string(logData))
		return 1
	}

	if res.StatusCode > 499 { // useful fuzz input
		panic("server error: " + res.Status + "\n" + string(logData))
		return 1
	}

	if strings.Contains(string(logData), "panic:") {
		panic("couper panic: " + string(logData))
		return 1
	}

	if runtime.NumGoroutine() > 100 {
		panic("goroutine leak")
		return 1
	}

	return 0
}
