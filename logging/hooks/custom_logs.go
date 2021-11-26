package hooks

import (
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ logrus.Hook = &CustomLogs{}

const customLogField = "custom"

type CustomLogs struct{}

func (c *CustomLogs) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *CustomLogs) Fire(entry *logrus.Entry) error {
	if entry.Context != nil {
		if t, exists := entry.Data["type"]; exists {
			switch t {
			case "couper_access":
				fire(entry, request.AccessLogFields, request.CustomLogsCtx)
			case "couper_backend":
				fire(entry, request.BackendLogFields, request.ContextType)
			}
		}
	}

	return nil
}

func fire(entry *logrus.Entry, bodyKey, ctxKey request.ContextKey) {
	var evalCtx *eval.Context

	if ctx, ok := entry.Context.Value(ctxKey).(*eval.Context); ok {
		evalCtx = ctx
	} else {
		evalCtx = eval.NewContext(nil, nil)
	}

	bodies := entry.Context.Value(bodyKey)
	if bodies == nil {
		return
	}

	hclBodies, ok := bodies.([]hcl.Body)
	if !ok {
		return
	}

	if fields := eval.ApplyCustomLogs(evalCtx.HCLContext(), hclBodies, entry); len(fields) > 0 {
		entry.Data[customLogField] = fields
	}
}
