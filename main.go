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

	runtimeConf := runtime.NewConfig(nil)
	set := flag.NewFlagSet("global", flag.ContinueOnError)
	set.StringVar(&runtimeConf.File, "f", runtimeConf.File, "-f ./couper.hcl")
	set.StringVar(&runtimeConf.LogFormat, "log-format", runtimeConf.LogFormat, "-log-format=common")
	err := set.Parse(args.Filter(set))
	if err != nil {
		logrus.WithFields(fields).Fatal(err)
	}
	envConf := &runtime.Config{}
	env.Decode(envConf)
	runtimeConf = runtimeConf.Merge(envConf)

	logger := newLogger(runtimeConf.LogFormat).WithFields(fields)

	wd, err := runtime.SetWorkingDirectory(runtimeConf.File)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Infof("working directory: %s", wd)

	gatewayConf, err := config.LoadFile(path.Base(runtimeConf.File))
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

	if format == "json" {
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
			logrus.FieldKeyMsg:  "message",
		}}
	}
	logger.Level = logrus.DebugLevel
	return logger
}
