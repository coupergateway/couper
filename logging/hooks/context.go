package hooks

import (
	"sync/atomic"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/logging"
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

	if field, ok := entry.Data["type"]; ok && field == beTypeField {
		if bytes, i := entry.Context.Value(request.BackendBytes).(*int64); i {
			response, r := entry.Data["response"].(logging.Fields)
			b := atomic.LoadInt64(bytes)
			if r && b > 0 {
				response["bytes"] = b
			}
		}
	}

	return nil
}
