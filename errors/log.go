package errors

import (
	"github.com/sirupsen/logrus"
)

var _ logrus.Hook = &LogHook{}

type LogHook struct{}

func (l *LogHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel}
}

func (l *LogHook) Fire(entry *logrus.Entry) error {
	err, exist := entry.Data[logrus.ErrorKey]
	if !exist {
		return nil
	}

	delete(entry.Data, logrus.ErrorKey)

	gerr, ok := err.(*Error)
	if !ok {
		return nil
	}

	if kinds := gerr.Kinds(); len(kinds) > 0 {
		entry.Data["error_type"] = kinds[0]
	}

	if len(entry.Message) > 0 {
		entry.Message += ": " + gerr.LogError()
	} else {
		entry.Message = gerr.LogError()
	}

	return nil
}
