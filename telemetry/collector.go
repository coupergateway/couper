package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var _ prometheus.Collector = &ServiceNameCollector{}

type ServiceNameCollector struct {
	labelPair *dto.LabelPair
	wrapped   prometheus.Collector
}

func NewServiceNameCollector(name string, coll prometheus.Collector) prometheus.Collector {
	serviceName := "service_name"
	return &ServiceNameCollector{
		labelPair: &dto.LabelPair{Name: &serviceName, Value: &name},
		wrapped:   coll,
	}
}

func (s *ServiceNameCollector) Describe(descs chan<- *prometheus.Desc) {
	s.wrapped.Describe(descs)
}

func (s *ServiceNameCollector) Collect(metrics chan<- prometheus.Metric) {
	metricsCh := make(chan prometheus.Metric, 10)
	go func() {
		s.wrapped.Collect(metricsCh)
		close(metricsCh)
	}()
	for m := range metricsCh {
		metrics <- &WrappedMetrics{
			inner:     m,
			labelPair: s.labelPair,
		}
	}
}

var _ prometheus.Metric = &WrappedMetrics{}

type WrappedMetrics struct {
	inner     prometheus.Metric
	labelPair *dto.LabelPair
}

func (w *WrappedMetrics) Desc() *prometheus.Desc {
	return w.inner.Desc()
}

func (w *WrappedMetrics) Write(metric *dto.Metric) error {
	if err := w.inner.Write(metric); err != nil {
		return err
	}
	metric.Label = append(metric.Label, w.labelPair)
	return nil
}
