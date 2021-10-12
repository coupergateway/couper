package server_test

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/internal/test"
)

func TestServeMetrics(t *testing.T) {
	helper := test.New(t)
	shutdown, _ := newCouper("testdata/integration/telemetry/01_couper.hcl", helper)
	defer shutdown()

	client := test.NewHTTPClient()
	mreq, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://localhost:%d/metrics", config.DefaultSettings.TelemetryMetricsPort), nil)
	helper.Must(err)

	clientReq, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	res, err := client.Do(clientReq)
	helper.Must(err)
	helper.Must(res.Body.Close())

	clientReq.URL.Path = "/notfound"
	res, err = client.Do(clientReq)
	helper.Must(err)
	helper.Must(res.Body.Close())

	res, err = client.Do(mreq)
	if err != nil {
		t.Fatalf("metrics endpoint could not be reached: %v", err)
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	// due to random client remote port
	result := regexp.MustCompile(`127\.0\.0\.1:\d{5}`).ReplaceAll(b, []byte("127.0.0.1"))
	defer func() {
		if t.Failed() {
			println(string(result))
		}
	}()

	expMetrics := []string{
		`couper_backend_request_duration_seconds_count{backend_name="anything",code="200",hostname="127.0.0.1",method="GET",origin="127.0.0.1",service_name="my-service",service_version="0"} 1`,
		`couper_backend_request_total{backend_name="anything",code="200",hostname="127.0.0.1",method="GET",origin="127.0.0.1",service_name="my-service",service_version="0"} 1`,
		`couper_client_request_duration_seconds_count{code="200",host="localhost:8080",method="GET",service_name="my-service",service_version="0"} 1`,
		`couper_client_request_duration_seconds_count{code="404",host="localhost:8080",method="GET",service_name="my-service",service_version="0"} 1`,
		`couper_client_request_total{code="200",host="localhost:8080",method="GET",service_name="my-service",service_version="0"} 1`,
		`couper_client_request_total{code="404",host="localhost:8080",method="GET",service_name="my-service",service_version="0"} 1`,
		`couper_client_request_error_types_total{error="route_not_found_error",service_name="my-service",service_version="0"} 1`,
		`couper_client_connections_total{service_name="my-service",service_version="0"} 2`,
		`go_goroutines{service_name="my-service"}`,
	}

	for _, expMetric := range expMetrics {
		if !strings.Contains(string(result), expMetric) {
			t.Errorf("missing metric: %s", expMetric)
		}
	}
}
