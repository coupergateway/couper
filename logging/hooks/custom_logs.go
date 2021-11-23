package hooks

import (
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ logrus.Hook = &CustomLogs{}

type CustomLogs struct{}

func (c *CustomLogs) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *CustomLogs) Fire(entry *logrus.Entry) error {
	if entry.Context != nil {
		if t, exists := entry.Data["type"]; exists {
			switch t {
			case "couper_access":
				fire(entry, request.AccessLogFields)
			case "couper_backend":
				fire(entry, request.BackendLogFields)
			}
		}
	}

	return nil
}

func fire(entry *logrus.Entry, key request.ContextKey) {
	var evalCtx *eval.Context

	if ctx, ok := entry.Context.Value(request.ContextType).(*eval.Context); ok {
		evalCtx = ctx
	} else {
		evalCtx = eval.NewContext(nil, nil)
	}

	if bodies := entry.Context.Value(key); bodies != nil {
		if fields := eval.ApplyCustomLogs(evalCtx, bodies.([]hcl.Body), entry); len(fields) > 0 {
			entry.Data["custom"] = fields
		}
	}
}
