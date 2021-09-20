package hooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/avenga/couper/telemetry/provider"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/telemetry/instrumentation"
)

var _ logrus.Hook = &Error{}

type Error struct{}

func (l *Error) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel, logrus.WarnLevel}
}

func (l *Error) Fire(entry *logrus.Entry) error {
	err, exist := entry.Data[logrus.ErrorKey]
	if !exist {
		return nil
	}

	delete(entry.Data, logrus.ErrorKey)

	gerr, ok := err.(*errors.Error)
	if !ok {
		entry.Message = errors.AppendMsg(entry.Message, fmt.Sprintf("%v", err))
		return nil
	}

	kind := strings.Replace(gerr.Error(), " ", "_", -1)
	if kinds := gerr.Kinds(); len(kinds) > 0 {
		entry.Data["error_type"] = kinds[0]
		kind = kinds[0]
	}

	meter := provider.Meter("couper/errors")
	counter := metric.Must(meter).
		NewInt64Counter(instrumentation.Prefix+"client_request_error_types_total",
			metric.WithDescription(string(unit.Dimensionless)),
		)
	meter.RecordBatch(context.Background(), []attribute.KeyValue{
		attribute.String("error", kind),
	},
		counter.Measurement(1),
	)

	entry.Message = errors.AppendMsg(entry.Message, gerr.LogError())

	return nil
}
