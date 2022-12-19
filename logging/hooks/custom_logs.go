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

	ctx := syncedUpstreamContext(evalCtx, entry)

	if fields := eval.ApplyCustomLogs(ctx, hclBodies, entry); len(fields) > 0 {
		entry.Data[customLogField] = fields
	}
}

func fireUpstream(entry *logrus.Entry) {
	evalCtx, ok := entry.Context.Value(request.ContextType).(*eval.Context)
	if !ok {
		return
	}
	bodyCh, _ := entry.Context.Value(request.LogCustomUpstream).(chan hcl.Body)
	if bodyCh == nil {
		return
	}

	var bodies []hcl.Body
	select {
	case body := <-bodyCh:
		bodies = append(bodies, body)
	case <-entry.Context.Done():
	}

	ctx := syncedUpstreamContext(evalCtx, entry)

	if fields := eval.ApplyCustomLogs(ctx, bodies, entry); len(fields) > 0 {
		entry.Data[customLogField] = fields
	}
}

// syncedUpstreamContext prepares the local backend variable.
func syncedUpstreamContext(evalCtx *eval.Context, entry *logrus.Entry) *hcl.EvalContext {
	ctx := evalCtx.HCLContextSync()

	tr, _ := entry.Context.Value(request.TokenRequest).(string)
	rtName, _ := entry.Context.Value(request.RoundTripName).(string)
	isTr := tr != ""

	if rtName == "" {
		return ctx
	}

	if _, ok := ctx.Variables[eval.BackendRequests]; ok {
		for k, v := range ctx.Variables[eval.BackendRequests].AsValueMap() {
			if isTr && k == eval.TokenRequestPrefix+tr {
				ctx.Variables[eval.BackendRequest] = v
				break
			} else if k == rtName {
				ctx.Variables[eval.BackendRequest] = v
				break
			}
		}
	}

	if _, ok := ctx.Variables[eval.BackendResponses]; ok {
		for k, v := range ctx.Variables[eval.BackendResponses].AsValueMap() {
			if isTr && k == eval.TokenRequestPrefix+tr {
				ctx.Variables[eval.BackendResponse] = v
				break
			} else if k == rtName {
				ctx.Variables[eval.BackendResponse] = v
				break
			}
		}
	}

	return ctx
}
