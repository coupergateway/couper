package telemetry

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	prompkg "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"

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
func InitExporter(ctx context.Context, opts *Options, logEntry *logrus.Entry) {
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
	}

	if opts.Traces {
		wg.Add(1)
		otel.Handle(initTraceExporter(ctx, opts, log, wg))
	}

	wg.Wait()
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

	hostname, err := os.Hostname()
	otel.Handle(err)

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.HostNameKey.String(hostname),
		semconv.ServiceNameKey.String(opts.ServiceName),
		semconv.ServiceVersionKey.String(utils.VersionName),
	)

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resources),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
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
		promExporter, err := newPromExporter(opts)
		if err != nil {
			return err
		}
		provider.SetMeterProvider(promExporter.MeterProvider())
		go func() {
			metrics := NewMetricsServer(log, promExporter, opts.MetricsPort)
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

	metricClient := otlpmetricgrpc.NewClient(clientOps...)
	metricExp, err := otlpmetric.New(ctx, metricClient)
	if err != nil {
		return err
	}

	pusher := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			metricExp,
		),
		controller.WithExporter(metricExp),
		controller.WithCollectPeriod(collectPeriod),
	)
	if err = pusher.Start(ctx); err != nil {
		return err
	}

	provider.SetMeterProvider(pusher.MeterProvider())

	go pushOnShutdown(ctx, pusher.Stop)

	log.Info("couper is pushing metrics")

	return nil
}

// pushOnShutdown pushes any last exports to the receiver.
func pushOnShutdown(ctx context.Context, shutdownFdn func(ctx context.Context) error) {
	<-ctx.Done()
	shtctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	otel.Handle(shutdownFdn(shtctx))
}

func newPromExporter(opts *Options) (*prometheus.Exporter, error) {
	config := prometheus.Config{
		Registry: prompkg.NewRegistry(),
	}

	config.Registry.MustRegister(NewServiceNameCollector(opts.ServiceName, collectors.NewGoCollector()))
	config.Registry.MustRegister(NewServiceNameCollector(opts.ServiceName, collectors.NewProcessCollector(
		collectors.ProcessCollectorOpts{
			Namespace: "couper", // name prefix
		},
	)))

	ctlr := controller.New(
		processor.New(
			selector.NewWithHistogramDistribution(),
			export.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opts.ServiceName),
			semconv.ServiceVersionKey.String(utils.VersionName),
		)),
	)
	promExporter, err := prometheus.New(config, ctlr)
	return promExporter, err
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
