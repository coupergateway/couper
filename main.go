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

func main() {
	fields := logrus.Fields{"type": "couper_daemon"}
	defaultLogger := newLogger(runtime.DefaultConfig).WithFields(fields)

	args := command.NewArgs()

	var configFile string
	set := flag.NewFlagSet("global", flag.ContinueOnError)
	set.StringVar(&configFile, "f", "couper.hcl", "-f ./couper.conf")
	err := set.Parse(args.Filter(set))
	if err != nil {
		defaultLogger.Fatal(err)
	}

	wd, err := runtime.SetWorkingDirectory(configFile)
	if err != nil {
		defaultLogger.Fatal(err)
	}

	gatewayConf, err := config.LoadFile(path.Base(configFile))
	if err != nil {
		defaultLogger.Fatal(err)
	}

	httpConf, err := runtime.NewHTTPConfig(gatewayConf, args)
	if err != nil {
		defaultLogger.Fatal(err)
	}

	logEntry := newLogger(httpConf).WithFields(fields)
	logEntry.Infof("working directory: %s", wd)

	entrypointHandlers := runtime.BuildEntrypointHandlers(gatewayConf, httpConf, logEntry)

	ctx := command.ContextWithSignal(context.Background())
	serverList, listenCmdShutdown := server.NewServerList(ctx, logEntry, httpConf, entrypointHandlers)
	for _, srv := range serverList {
		srv.Listen()
	}
	listenCmdShutdown()
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
