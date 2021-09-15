package server_test

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestServeMetrics(t *testing.T) {
	helper := test.New(t)
	shutdown, _ := newCouper("testdata/integration/telemetry/01_couper.hcl", helper)
	defer shutdown()

	client := test.NewHTTPClient()
	mreq, err := http.NewRequest(http.MethodGet, "http://localhost:9090/metrics", nil)
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
	helper.Must(err)

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	// due to random client remote port
	result := regexp.MustCompile(`127\.0\.0\.1:\d{5}`).ReplaceAll(b, []byte("127.0.0.1"))

	expMetrics := []string{
		`backend_request_duration_seconds_count{backend_name="anything",hostname="127.0.0.1",origin="127.0.0.1",response_status="200",service_name="couper",service_version="0"} 1`,
		`backend_request_total{backend_name="anything",hostname="127.0.0.1",origin="127.0.0.1",response_status="200",service_name="couper",service_version="0"} 1`,
		`client_request_duration_seconds_count{host="localhost:8080",response_status="200",service_name="couper",service_version="0"} 1`,
		`client_request_total{host="localhost:8080",response_status="200",service_name="couper",service_version="0"} 1`,
		`client_request_total{host="localhost:8080",response_status="404",service_name="couper",service_version="0"} 1`,
		`connections_count{backend="anything",host="127.0.0.1",origin="127.0.0.1",service_name="couper",service_version="0"} 1`,
		`connections_total{backend="anything",host="127.0.0.1",origin="127.0.0.1",service_name="couper",service_version="0"} 1`,
		`client_request_route_not_found_error_total{error="true",host="localhost:8080",service_name="couper",service_version="0"} 1`,
	}

	for _, expMetric := range expMetrics {
		if !strings.Contains(string(result), expMetric) {
			t.Errorf("missing metric: %s", expMetric)
		}
	}
}
