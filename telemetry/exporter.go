package telemetry

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"google.golang.org/grpc"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/telemetry/provider"
	"github.com/avenga/couper/utils"
)

const (
	ExporterInvalid uint8 = iota
	ExporterPrometheus
	ExporterJaeger
	ExporterOTLP
	ExporterStdout
)

const otlpExporterEnvKey = "OTEL_EXPORTER_OTLP_ENDPOINT"

// InitExporter initialises configured metrics and/or trace exporter.
func InitExporter(ctx context.Context, opts *Options, memStore *cache.MemoryStore, logEntry *logrus.Entry) error {
	log := logEntry.WithField("type", "couper_telemetry")
	otel.SetErrorHandler(ErrorHandleFunc(func(e error) { // configure otel to use our logger for error handling
		if e != nil {
			log.WithError(e).Error()
		}
	}))

	wg := &sync.WaitGroup{}
	if opts.Metrics {
		wg.Add(1)
		otel.Handle(initMetricExporter(ctx, opts, log, wg))

		if err := newBackendsObserver(memStore); err != nil {
			return err
		}
	}

	if opts.Traces {
		wg.Add(1)
		otel.Handle(initTraceExporter(ctx, opts, log, wg))
	}

	wg.Wait()

	return nil
}

func initTraceExporter(ctx context.Context, opts *Options, log *logrus.Entry, wg *sync.WaitGroup) error {
	defer wg.Done()

	endpoint := opts.TracesEndpoint
	if ep := os.Getenv(otlpExporterEnvKey); ep != "" {
		endpoint = ep
	}

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
		otlptracegrpc.WithEndpoint(endpoint),
	)

	withCancel, cancelFn := context.WithDeadline(ctx, time.Now().Add(time.Second))
	defer cancelFn()
	traceExp, err := otlptrace.New(withCancel, traceClient)
	if err != nil {
		return err
	}

	resources := newResource(opts.ServiceName)

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resources),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to TraceContext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	go pushOnShutdown(ctx, traceExp.Shutdown)

	log.WithField("endpoint", endpoint).Info("couper is pushing traces")

	return nil

}

func initMetricExporter(ctx context.Context, opts *Options, log *logrus.Entry, wg *sync.WaitGroup) error {
	defer wg.Done()

	exporter := parseExporter(opts.MetricsExporter)
	if exporter == ExporterInvalid {
		return fmt.Errorf("metrics: unknown exporter: %s", opts.MetricsExporter)
	}

	if exporter == ExporterPrometheus {
		promExporter, promRegistry, err := newPromExporter(opts)
		if err != nil {
			return err
		}

		meterProvider := metric.NewMeterProvider(
			metric.WithResource(newResource(opts.ServiceName)),
			metric.WithReader(promExporter),
		)
		provider.SetMeterProvider(meterProvider)

		go func() {
			metrics := NewMetricsServer(log, promRegistry, opts.MetricsPort)
			go metrics.ListenAndServe()
			<-ctx.Done()
			otel.Handle(metrics.Close())
		}()

		return nil
	}

	if exporter != ExporterOTLP {
		return fmt.Errorf("metrics: unsupported exporter: %s", opts.MetricsExporter)
	}

	endpoint := opts.TracesEndpoint
	if ep := os.Getenv(otlpExporterEnvKey); ep != "" {
		endpoint = ep
	}

	clientOps := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithReconnectionPeriod(time.Second * 5),
	}

	collectPeriod := opts.MetricsCollectPeriod
	if collectPeriod.Milliseconds() == 0 {
		collectPeriod = time.Second * 2
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, clientOps...)
	if err != nil {
		return err
	}

	periodicReader := metric.NewPeriodicReader(
		metricExporter,
		metric.WithInterval(collectPeriod),
	)

	res := newResource(opts.ServiceName)

	meterProvider :=
		metric.NewMeterProvider(
			metric.WithResource(res),
			metric.WithReader(periodicReader),
		)
	provider.SetMeterProvider(meterProvider)

	go pushOnShutdown(ctx, periodicReader.Shutdown)

	log.Info("couper is pushing metrics", endpoint)

	return nil
}

// pushOnShutdown pushes any last exports to the receiver.
func pushOnShutdown(ctx context.Context, shutdownFdn func(ctx context.Context) error) {
	<-ctx.Done()
	shtctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	otel.Handle(shutdownFdn(shtctx))
}

func newPromExporter(opts *Options) (*otelprom.Exporter, *WrappedRegistry, error) {
	strPtr := func(s string) *string { return &s }

	registry := NewWrappedRegistry(prom.NewRegistry(), &dto.LabelPair{
		Name:  strPtr("service_name"),
		Value: strPtr(opts.ServiceName),
	}, &dto.LabelPair{
		Name:  strPtr("service_version"),
		Value: strPtr(utils.VersionName),
	})

	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(
		collectors.ProcessCollectorOpts{
			Namespace: "couper", // name prefix
		},
	))

	promExporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
		otelprom.WithoutScopeInfo(),
		otelprom.WithoutTargetInfo(),
	)

	return promExporter, registry, err
}

func newResource(serviceName string) *resource.Resource {
	hostname, err := os.Hostname()
	otel.Handle(err)

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.HostNameKey.String(hostname),
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(utils.VersionName),
	)
}

func parseExporter(e string) uint8 {
	switch e {
	case "prometheus":
		return ExporterPrometheus
	case "jaeger":
		return ExporterJaeger
	case "otlp":
		return ExporterOTLP
	default:
		return ExporterInvalid
	}
}
