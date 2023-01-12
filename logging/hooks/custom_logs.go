package hooks

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

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
	logValues, _ := entry.Context.Value(request.LogCustomUpstreamValues).(*[]cty.Value)
	logErrors, _ := entry.Context.Value(request.LogCustomUpstreamErrors).(*[]error)

	if fields := eval.MergeCustomLogs(*logValues, *logErrors, entry); len(fields) > 0 {
		entry.Data[customLogField] = fields
	}
}
