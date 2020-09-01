//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/config/runtime"
	"go.avenga.cloud/couper/gateway/server"
)

var configFile = flag.String("f", "couper.hcl", "-f ./couper.conf")

func main() {
	httpConf := runtime.NewHTTPConfig()
	logger := newLogger(httpConf)
	if err := runtime.SetWorkingDirectory(*configFile); err != nil {
		logger.Fatal(err)
	}

	wd, _ := os.Getwd()
	logger.WithField("working-directory", wd).Info()

	gatewayConf, err := config.LoadFile(path.Base(*configFile))
	if err != nil {
		logger.Fatal(err)
	}

	entrypointHandlers := runtime.BuildEntrypointHandlers(gatewayConf, httpConf, logger)

	ctx := command.ContextWithSignal(context.Background())
	for _, srv := range server.NewServerList(ctx, logger, httpConf, entrypointHandlers) {
		srv.Listen()
	}
	<-ctx.Done() // TODO: shutdown deadline
}

func newLogger(conf *runtime.HTTPConfig) *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout
	if conf.LogFormat == "json" {
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
			logrus.FieldKeyMsg:  "message",
		}}
	}
	logger.Level = logrus.DebugLevel
	return logger.WithField("type", "couper")
}
