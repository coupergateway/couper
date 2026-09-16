package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

// serveWithMetrics serves one request through handler and returns the exported bucket
// boundaries of the client request duration histogram and their cumulative counts.
func serveWithMetrics(t *testing.T, handler http.Handler) ([]float64, []uint64) {
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

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != instrumentation.ClientRequestDuration {
			continue
		}

		var bounds []float64
		var counts []uint64
		for _, bucket := range family.GetMetric()[0].GetHistogram().GetBucket() {
			bounds = append(bounds, bucket.GetUpperBound())
			counts = append(counts, bucket.GetCumulativeCount())
		}

		return bounds, counts
	}

	t.Fatalf("%s not exported", instrumentation.ClientRequestDuration)

	return nil, nil
}

func TestMetricsHandler_DefaultBucketBoundaries(t *testing.T) {
	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})

	bounds, _ := serveWithMetrics(t, NewMetricsHandler()(inner))

	want := instrumentation.DefaultDurationSecondsBoundaries
	if len(bounds) != len(want) {
		t.Fatalf("want %d boundaries, got %d: %v", len(want), len(bounds), bounds)
	}
	for i, boundary := range want {
		if bounds[i] != boundary {
			t.Errorf("boundary %d: want %v, got %v", i, boundary, bounds[i])
		}
	}
}

// The SDK's default boundaries put every request faster than 5s into one bucket.
func TestMetricsHandler_ResolvesSubSecondDurations(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	// The handler advances the clock instead of sleeping, so the recorded duration
	// is exactly 25ms on every machine.
	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		now = now.Add(25 * time.Millisecond)
		rw.WriteHeader(http.StatusOK)
	})

	bounds, counts := serveWithMetrics(t, &MetricsHandler{handler: inner, clock: clock})

	capturedAt := -1
	for i, count := range counts {
		if count > 0 {
			capturedAt = i
			break
		}
	}

	if capturedAt < 0 {
		t.Fatalf("observation was not recorded: %v", counts)
	}

	const want = 0.025
	if bounds[capturedAt] != want {
		t.Errorf("a 25ms request was first captured at le=%v, want le=%v", bounds[capturedAt], want)
	}
}
