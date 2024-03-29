package hooks

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/env"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
	"github.com/coupergateway/couper/logging"
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
	return []logrus.Level{
		logrus.PanicLevel, // reasonable?
		logrus.FatalLevel, // reasonable?
		logrus.ErrorLevel,
		logrus.WarnLevel, // not used?
		logrus.InfoLevel,
	}
}

func (c *CustomLogs) Fire(entry *logrus.Entry) error {
	if entry.Context != nil {
		if t, exists := entry.Data["type"]; exists {
			switch t {
			case "couper_job":
				fallthrough
			case acTypeField:
				fireAccess(entry)
			case beTypeField:
				fireUpstream(entry)
			}
		}
	}

	return nil
}

func fireAccess(entry *logrus.Entry) {
	evalCtx, ok := entry.Context.Value(request.ContextType).(*eval.Context)
	if !ok {
		return
	}

	bodies := entry.Context.Value(request.LogCustomAccess)
	if bodies == nil {
		return
	}

	hclBodies, ok := bodies.([]hcl.Body)
	if !ok {
		return
	}

	if fields := eval.ApplyCustomLogs(evalCtx.HCLContextSync(), hclBodies, entry); len(fields) > 0 {
		entry.Data[customLogField] = fields
	}
}

func fireUpstream(entry *logrus.Entry) {
	logError, _ := entry.Context.Value(request.LogCustomUpstreamError).(*error)
	if *logError != nil {
		entry.Debug(*logError)
		return
	}
	logValue, _ := entry.Context.Value(request.LogCustomUpstreamValue).(*cty.Value)
	fields := seetie.ValueToLogFields(*logValue)
	entry.Data[customLogField] = fields
}
