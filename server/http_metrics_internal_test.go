package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/config/runtime"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

// Covers the attribute reaching the exported histogram through server.New, which the
// NewMetricsHandler unit tests do not exercise.
func TestServerAppliesConfiguredDurationBuckets(t *testing.T) {
	for _, tt := range []struct {
		name    string
		setting string
		want    []float64
	}{
		{
			name: "default boundaries",
			want: instrumentation.DefaultDurationSecondsBoundaries,
		},
		{
			name:    "configured boundaries are sorted",
			setting: `beta_metrics_request_duration_buckets = [1, 0.05, 10, 0.2]`,
			want:    []float64{0.05, 0.2, 1, 10},
		},
	} {
		t.Run(tt.name, func(subT *testing.T) {
			hcl := fmt.Sprintf(`
				server "metrics" {
				  endpoint "/" {
				    response {
				      status = 204
				    }
				  }
				}
				settings {
				  beta_metrics = true
				  %s
				}
			`, tt.setting)

			conf, err := configload.LoadBytes([]byte(hcl), "couper.hcl")
			if err != nil {
				subT.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conf.Context = conf.Context.Value(request.ContextType).(*eval.Context).WithContext(ctx)

			log := logrus.New()
			log.Out = io.Discard
			logger := log.WithContext(ctx)
			srvConf, err := runtime.NewServerConfiguration(conf, logger, cache.New(logger, ctx.Done()))
			if err != nil {
				subT.Fatal(err)
			}

			timings := runtime.DefaultTimings
			servers, _, err := NewServers(ctx, conf.Context, logger, conf.Settings, &timings, srvConf)
			if err != nil {
				subT.Fatal(err)
			}
			if len(servers) != 1 {
				subT.Fatalf("want one server, got %d", len(servers))
			}

			bounds := serveAndGatherBounds(subT, servers[0].srv.Handler)

			if len(bounds) != len(tt.want) {
				subT.Fatalf("want %d boundaries, got %d: %v", len(tt.want), len(bounds), bounds)
			}
			for i, boundary := range tt.want {
				if bounds[i] != boundary {
					subT.Errorf("boundary %d: want %v, got %v", i, boundary, bounds[i])
				}
			}
		})
	}
}

// serveAndGatherBounds serves one request and returns the exported bucket boundaries of
// the client request duration histogram.
func serveAndGatherBounds(t *testing.T, handler http.Handler) []float64 {
	t.Helper()

	registry := prom.NewRegistry()
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
		otelprom.WithoutScopeInfo(),
		otelprom.WithoutTargetInfo(),
	)
	if err != nil {
		t.Fatal(err)
	}

	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	provider.SetMeterProvider(meterProvider)
	t.Cleanup(func() {
		_ = meterProvider.Shutdown(context.Background())
		provider.SetMeterProvider(noop.NewMeterProvider())
	})

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != instrumentation.ClientRequestDuration {
			continue
		}

		var bounds []float64
		for _, bucket := range family.GetMetric()[0].GetHistogram().GetBucket() {
			bounds = append(bounds, bucket.GetUpperBound())
		}

		return bounds
	}

	t.Fatalf("%s not exported", instrumentation.ClientRequestDuration)

	return nil
}
