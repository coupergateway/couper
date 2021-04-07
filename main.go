//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/logging"
)

var (
	fields = logrus.Fields{
		"build":   runtime.BuildName,
		"version": runtime.VersionName,
	}
	hook logrus.Hook
)

func main() {
	logrus.Exit(realmain(os.Args))
}

func realmain(arguments []string) int {
	args := command.NewArgs(arguments)
	ctx := context.Background()

	if len(args) == 0 || command.NewCommand(ctx, args[0]) == nil {
		command.Help()
		return 1
	}

	cmd := args[0]
	args = args[1:]

	if cmd == "version" { // global options are not required, fast exit.
		_ = command.NewCommand(ctx, cmd).Execute(args, nil, nil)
		return 0
	}

	var filePath, logFormat string
	var logPretty bool
	var fileWatch bool
	set := flag.NewFlagSet("global", flag.ContinueOnError)
	set.StringVar(&filePath, "f", config.DefaultFilename, "-f ./couper.hcl")
	set.StringVar(&logFormat, "log-format", config.DefaultSettings.LogFormat, "-log-format=common")
	set.BoolVar(&logPretty, "log-pretty", config.DefaultSettings.LogPretty, "-log-pretty")
	set.BoolVar(&fileWatch, "watch", fileWatch, "-watch")
	err := set.Parse(args.Filter(set))
	if err != nil {
		newLogger(logFormat, logPretty).Error(err)
		return 1
	}

	confFile, err := configload.LoadFile(filePath)
	if err != nil {
		newLogger(logFormat, logPretty).Error(err)
		return 1
	}

	// The file gets initialized with the default settings, flag args are preferred over file settings.
	// Only override file settings if the flag value differ from the default.
	if logFormat != config.DefaultSettings.LogFormat {
		confFile.Settings.LogFormat = logFormat
	}
	if logPretty != config.DefaultSettings.LogPretty {
		confFile.Settings.LogPretty = logPretty
	}
	logger := newLogger(confFile.Settings.LogFormat, confFile.Settings.LogPretty)

	wd, err := os.Getwd()
	if err != nil {
		logger.Error(err)
		return 1
	}
	logger.Infof("working directory: %s", wd)

	if fileWatch {
		logger.WithFields(fields).Info("watching configuration file")
		errCh := make(chan error, 1)
		reloadCh := make(chan struct{}, 1)
		watchContext, cancelFn := context.WithCancel(ctx)
		defer cancelFn()

		go func() {
			errCh <- command.NewCommand(watchContext, cmd).Execute(args, confFile, logger)
		}()

		go func() {
			ticker := time.NewTicker(time.Second / 4)
			defer ticker.Stop()
			var lastChange time.Time
			for {
				<-ticker.C
				fileInfo, fileErr := os.Stat(confFile.Filename)
				if fileErr != nil {
					logger.WithFields(fields).Error(fileErr)
					continue
				}

				if lastChange.IsZero() { // first round
					lastChange = fileInfo.ModTime()
					continue
				}

				if fileInfo.ModTime().After(lastChange) {
					reloadCh <- struct{}{}
				}
				lastChange = fileInfo.ModTime()
			}
		}()
		for {
			select {
			case err = <-errCh:
				if err != nil {
					logger.Error(err)
					return 1
				}
				return 0
			case <-reloadCh:
				logger.WithFields(fields).Info("reloading couper configuration")
				cf, reloadErr := configload.LoadFile(confFile.Filename) // we are at wd, just filename
				if reloadErr != nil {
					logger.WithFields(fields).Errorf("reload failed: %v", reloadErr)
					continue
				}
				confFile = cf
				cancelFn()                                       // (hard) shutdown running couper
				<-errCh                                          // drain current error due to cancel and ensure closed ports
				watchContext, cancelFn = context.WithCancel(ctx) // replace previous pair
				go func() {
					errCh <- command.NewCommand(watchContext, cmd).Execute(args, confFile, logger)
				}()
			}
		}
	} else {
		if err = command.NewCommand(ctx, cmd).Execute(args, confFile, logger); err != nil {
			logger.Error(err)
			return 1
		}
	}
	return 0
}

// newLogger creates a log instance with the configured formatter.
// Since the format option may required to be correct in early states
// we parse the env configuration on every call.
func newLogger(format string, pretty bool) *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout
	if hook != nil {
		logger.AddHook(hook)
		logger.Out = ioutil.Discard
	}

	settings := &config.Settings{
		LogFormat: format,
		LogPretty: pretty,
	}
	env.Decode(settings)

	logConf := &logging.Config{
		TypeFieldKey: "couper_daemon",
	}
	env.Decode(logConf)

	if settings.LogFormat == "json" {
		logger.SetFormatter(logging.NewJSONColorFormatter(logConf.ParentFieldKey, settings.LogPretty))
	}
	logger.Level = logrus.DebugLevel
	return logger.WithField("type", logConf.TypeFieldKey).WithFields(fields)
}
