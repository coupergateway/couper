package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/logging"
)

func NewWrappedHandler(log *logrus.Entry, handler http.Handler) http.Handler {
	accessLog := logging.NewAccessLog(nil, log)

	uidHandler := middleware.NewUIDHandler(&config.DefaultSettings, "")(handler)
	logHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r := req.WithContext(context.WithValue(req.Context(), request.LogDebugLevel, true))
		accessLog.ServeHTTP(rw, r, uidHandler, time.Now())
	})
	return logHandler
}
