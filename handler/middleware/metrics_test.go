package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

// serveWithMetrics returns the exported bucket boundaries and their cumulative counts.
func serveWithMetrics(t *testing.T, handlerDelay time.Duration) ([]float64, []uint64) {
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
	provider.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter)))

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		time.Sleep(handlerDelay)
		rw.WriteHeader(http.StatusOK)
	})
	NewMetricsHandler()(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

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
	bounds, _ := serveWithMetrics(t, 0)

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
	bounds, counts := serveWithMetrics(t, 25*time.Millisecond)

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

	if bounds[capturedAt] > 0.1 {
		t.Errorf("a 25ms request was first captured at le=%v; the boundaries do not resolve sub-second durations", bounds[capturedAt])
	}
}
