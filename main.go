//go:generate go run assets/generate/generate.go

package main

import (
	"flag"
	"os"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
)

func main() {
	fields := logrus.Fields{"type": "couper_daemon"}

	args := command.NewArgs()
	if len(args) == 0 || command.NewCommand(args[0]) == nil {
		command.Help()
		os.Exit(1)
	}

	// TODO: 'global' conf obj in runtime package
	var configFile, logFormat string
	set := flag.NewFlagSet("global", flag.ContinueOnError)
	set.StringVar(&configFile, "f", "couper.hcl", "-f ./couper.conf")
	set.StringVar(&logFormat, "log-format", "common", "-log-format=common")
	err := set.Parse(args.Filter(set))
	if err != nil {
		logrus.WithFields(fields).Fatal(err)
	}

	logger := newLogger(logFormat).WithFields(fields)

	wd, err := runtime.SetWorkingDirectory(configFile)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Infof("working directory: %s", wd)

	gatewayConf, err := config.LoadFile(path.Base(configFile))
	if err != nil {
		logger.Fatal(err)
	}

	var exitCode int
	if err = command.NewCommand(args[0]).Execute(args, gatewayConf, logger); err != nil {
		logger.Error(err)
		exitCode = 1
	}
	logrus.Exit(exitCode)
}

func newLogger(format string) logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = os.Stdout

	if envFormat := os.Getenv(env.PREFIX + "LOG_FORMAT"); envFormat != "" {
		format = envFormat
	}

	if format == "json" {
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
			logrus.FieldKeyMsg:  "message",
		}}
	}
	logger.Level = logrus.DebugLevel
	return logger
}
