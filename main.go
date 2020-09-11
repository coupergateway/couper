//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/server"
)

var configFile = flag.String("f", "couper.hcl", "-f ./couper.conf")

func main() {
	httpConf := runtime.NewHTTPConfig()
	logger := newLogger(httpConf)
	logEntry := logger.WithField("type", "couper_daemon")
	if err := runtime.SetWorkingDirectory(*configFile); err != nil {
		logEntry.Fatal(err)
	}

	wd, _ := os.Getwd()
	logEntry.Infof("working directory: %s", wd)

	gatewayConf, err := config.LoadFile(path.Base(*configFile))
	if err != nil {
		logEntry.Fatal(err)
	}

	entrypointHandlers := runtime.BuildEntrypointHandlers(gatewayConf, httpConf, logEntry)

	ctx := command.ContextWithSignal(context.Background())
	for _, srv := range server.NewServerList(ctx, logEntry, httpConf, entrypointHandlers) {
		srv.Listen()
	}
	<-ctx.Done() // TODO: shutdown deadline
}

func newLogger(conf *runtime.HTTPConfig) logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = os.Stdout
	if conf.LogFormat == "json" {
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
			logrus.FieldKeyMsg:  "message",
		}}
	}
	logger.Level = logrus.DebugLevel
	return logger
}
