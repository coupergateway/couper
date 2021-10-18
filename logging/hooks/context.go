package hooks

import (
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
)

var _ logrus.Hook = &Context{}

type Context struct{}

func (c *Context) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *Context) Fire(entry *logrus.Entry) error {
	_, exist := entry.Data["uid"]
	if entry.Context != nil && !exist {
		if uid := entry.Context.Value(request.UID); uid != nil {
			entry.Data["uid"] = uid
		}
	}
	return nil
}
