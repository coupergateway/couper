//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"os"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/config/runtime"
	"go.avenga.cloud/couper/gateway/server"
)

func main() {
	config := *runtime.DefaultConfig

	flag.StringVar(&config.ConfigFile, "f", runtime.DefaultConfig.ConfigFile, "-f ./couper.conf")
	flag.IntVar(&config.ListenPort, "p", runtime.DefaultConfig.ListenPort, "-p 8080")
	flag.BoolVar(&config.UseXFH, "xfh", runtime.DefaultConfig.UseXFH, "-xfh")

	if !flag.Parsed() {
		flag.Parse()
	}

	logger := newLogger()
	runtime.Configure(&config, logger)

	ctx := command.ContextWithSignal(context.Background())
	for _, srv := range server.NewServerList(ctx, logger, &config) {
		srv.Listen()
	}
	<-ctx.Done() // TODO: shutdown deadline
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
