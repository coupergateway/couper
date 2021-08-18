package logging

import (
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
)

var _ logrus.Hook = &ContextHook{}

type ContextHook struct {
}

func (c ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c ContextHook) Fire(entry *logrus.Entry) error {
	_, exist := entry.Data["uid"]
	if entry.Context != nil && !exist {
		if uid := entry.Context.Value(request.UID); uid != nil {
			entry.Data["uid"] = uid
		}
	}
	return nil
}
