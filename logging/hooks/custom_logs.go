package hooks

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/logging"
)

var (
	_ logrus.Hook = &CustomLogs{}

	acTypeField string
	beTypeField string
)

const customLogField = "custom"

type CustomLogs struct{}

func init() {
	logConf := *logging.DefaultConfig
	logConf.TypeFieldKey = "couper_access"
	env.DecodeWithPrefix(&logConf, "ACCESS_")
	acTypeField = logConf.TypeFieldKey

	logConf = *logging.DefaultConfig
	logConf.TypeFieldKey = "couper_backend"
	env.DecodeWithPrefix(&logConf, "BACKEND_")
	beTypeField = logConf.TypeFieldKey
}

func (c *CustomLogs) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *CustomLogs) Fire(entry *logrus.Entry) error {
	if entry.Context != nil {
		if t, exists := entry.Data["type"]; exists {
			switch t {
			case acTypeField:
				fire(entry, request.LogCustomAccess)
			case beTypeField:
				fire(entry, request.LogCustomUpstream)
			}
		}
	}

	return nil
}

func fire(entry *logrus.Entry, bodyKey request.ContextKey) {
	var evalCtx *eval.Context

	customEvalCtxCh, ok := entry.Context.Value(request.LogCustomEvalResult).(chan *eval.Context)
	if ok {
		select {
		case evalCtx = <-customEvalCtxCh:
		default: // pass through, e.g. on early errors we will not receive something useful
		}
	}

	if evalCtx == nil {
		evalCtx, ok = entry.Context.Value(request.ContextType).(*eval.Context)
		if !ok {
			return
		}
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