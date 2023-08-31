package hooks

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
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
	entry.Message = errors.AppendMsg(entry.Message, gerr.LogError())

	if entry.Data["type"] != acTypeField {
		return nil
	}

	meter := provider.Meter("couper/errors")

	counter, _ := meter.Int64Counter(
		instrumentation.Prefix+"client_request_error_types",
		instrument.WithDescription(string(unit.Dimensionless)),
	)

	counter.Add(entry.Context, 1, attribute.String("error", kind))

	return nil
}
