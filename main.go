package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

func main() {
	// TODO: command / args

	logger := newLogger()

	exampleConf := config.Load("example.hcl", logger)

	srv := server.New(command.ContextWithSignal(context.Background()), logger, exampleConf)
	os.Exit(srv.Listen())
}

func newLogger() *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
		logrus.FieldKeyTime: "timestamp",
		logrus.FieldKeyMsg:  "message",
	}}
	logger.Level = logrus.DebugLevel
	return logger.WithField("type", "couper")
}
