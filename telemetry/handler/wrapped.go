package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/handler/middleware"
	"github.com/coupergateway/couper/logging"
)

func NewWrappedHandler(log *logrus.Entry, handler http.Handler) http.Handler {
	accessLog := logging.NewAccessLog(nil, log)

	uidHandler := middleware.NewUIDHandler(config.NewDefaultSettings(), "")(handler)
	logHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := context.WithValue(req.Context(), request.LogDebugLevel, true)
		ctx = context.WithValue(ctx, request.StartTime, time.Now())
		var logStack *logging.Stack
		ctx, logStack = logging.NewStack(ctx)
		r := req.WithContext(ctx)
		uidHandler.ServeHTTP(rw, r)
		accessLog.Do(rw, r)
		logStack.Fire()
	})
	return logHandler
}
