//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/logging"
)

var (
	fields = logrus.Fields{
		"build":   runtime.BuildName,
		"version": runtime.VersionName,
	}
	testHook logrus.Hook
)

func main() {
	logrus.Exit(realmain(os.Args))
}

func realmain(arguments []string) int {
	args := command.NewArgs(arguments)
	ctx := context.Background()

	type globalFlags struct {
		FilePath            string        `env:"file"`
		FileWatch           bool          `env:"watch"`
		FileWatchRetryDelay time.Duration `env:"watch_retry_delay"`
		FileWatchRetries    int           `env:"watch_retries"`
		LogFormat           string        `env:"log_format"`
		LogPretty           bool          `env:"log_pretty"`
	}
	var flags globalFlags

	set := flag.NewFlagSet("global options", flag.ContinueOnError)
	set.StringVar(&flags.FilePath, "f", config.DefaultFilename, "-f ./my-path/couper.hcl")
	set.BoolVar(&flags.FileWatch, "watch", false, "-watch")
	set.DurationVar(&flags.FileWatchRetryDelay, "watch-retry-delay", time.Millisecond*500, "-watch-retry-delay 1s")
	set.IntVar(&flags.FileWatchRetries, "watch-retries", 5, "-watch-retries 10")
	set.StringVar(&flags.LogFormat, "log-format", config.DefaultSettings.LogFormat, "-log-format=json")
	set.BoolVar(&flags.LogPretty, "log-pretty", config.DefaultSettings.LogPretty, "-log-pretty")

	if len(args) == 0 || command.NewCommand(ctx, args[0]) == nil {
		command.Synopsis()
		set.Usage()
		return 1
	}

	cmd := args[0]
	args = args[1:]

	if cmd != "run" { // global options are not required atm, fast exit.
		err := command.NewCommand(ctx, cmd).Execute(args, nil, nil)
		if err != nil {
			set.Usage()
			color.Red("\n%v", err)
			return 1
		}
		return 0
	}

	err := set.Parse(args.Filter(set))
	if err != nil {
		newLogger(flags.LogFormat, flags.LogPretty).Error(err)
		return 1
	}
	env.Decode(&flags)

	confFile, err := configload.LoadFile(flags.FilePath)
	if err != nil {
		newLogger(flags.LogFormat, flags.LogPretty).WithError(err).Error()
		return 1
	}

	// The file gets initialized with the default settings, flag args are preferred over file settings.
	// Only override file settings if the flag value differ from the default.
	if flags.LogFormat != config.DefaultSettings.LogFormat {
		confFile.Settings.LogFormat = flags.LogFormat
	}
	if flags.LogPretty != config.DefaultSettings.LogPretty {
		confFile.Settings.LogPretty = flags.LogPretty
	}
	logger := newLogger(confFile.Settings.LogFormat, confFile.Settings.LogPretty)

	wd, err := os.Getwd()
	if err != nil {
		logger.Error(err)
		return 1
	}
	logger.Infof("working directory: %s", wd)

	if !flags.FileWatch {
		if err = command.NewCommand(ctx, cmd).Execute(args, confFile, logger); err != nil {
			logger.WithError(err).Error()
			return 1
		}
		return 0
	}

	logger.WithField("watch", logrus.Fields{
		"retry-delay": flags.FileWatchRetryDelay.String(),
		"max-retries": flags.FileWatchRetries,
	}).Info("watching configuration file")
	errCh := make(chan error, 1)
	errRetries := 0

	execCmd, restartSignal := newRestartableCommand(ctx, cmd)
	go func() {
		errCh <- execCmd.Execute(args, confFile, logger)
	}()

	reloadCh := watchConfigFile(confFile.Filename, logger, flags.FileWatchRetries, flags.FileWatchRetryDelay)
	for {
		select {
		case err = <-errCh:
			if err != nil {
				if netErr, ok := err.(*net.OpError); ok {
					if netErr.Op == "listen" && errRetries < flags.FileWatchRetries {
						errRetries++
						logger.Errorf("retry %d/%d due to listen error: %v", errRetries, flags.FileWatchRetries, netErr)

						// configuration load succeeded at this point, just restart the command
						execCmd, restartSignal = newRestartableCommand(ctx, cmd) // replace previous pair
						time.Sleep(flags.FileWatchRetryDelay)

						go func() {
							errCh <- execCmd.Execute(args, confFile, logger)
						}()
						continue
					} else if errRetries >= flags.FileWatchRetries {
						logger.Errorf("giving up after %d retries: %v", errRetries, netErr)
						return 1
					}
				}
				logger.WithError(err).Error()
				return 1
			}
			return 0
		case _, more := <-reloadCh:
			if !more {
				return 1
			}
			errRetries = 0 // reset
			logger.Info("reloading couper configuration")

			cf, reloadErr := configload.LoadFile(confFile.Filename) // we are at wd, just filename
			if reloadErr != nil {
				logger.WithError(reloadErr).Error("reload failed")
				time.Sleep(flags.FileWatchRetryDelay)
				continue
			}
			// dry run configuration
			_, reloadErr = runtime.NewServerConfiguration(cf, logger.WithFields(fields), nil)
			if reloadErr != nil {
				logger.WithError(reloadErr).Error("reload failed")
				time.Sleep(flags.FileWatchRetryDelay)
				continue
			}

			confFile = cf
			restartSignal <- struct{}{}                              // shutdown running couper
			<-errCh                                                  // drain current error due to cancel and ensure closed ports
			execCmd, restartSignal = newRestartableCommand(ctx, cmd) // replace previous pair
			go func() {
				// logger settings update gets ignored at this point
				// have to be locked for an update, skip this feature for now
				errCh <- execCmd.Execute(args, confFile, logger)
			}()
		}
	}
}

// newLogger creates a log instance with the configured formatter.
// Since the format option may required to be correct in early states
// we parse the env configuration on every call.
func newLogger(format string, pretty bool) *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout

	logger.AddHook(&errors.LogHook{})

	if testHook != nil {
		logger.AddHook(testHook)
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

func watchConfigFile(name string, logger logrus.FieldLogger, maxRetries int, retryDelay time.Duration) <-chan struct{} {
	reloadCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second / 4)
		defer ticker.Stop()
		var lastChange time.Time
		var errors int
		for {
			<-ticker.C
			fileInfo, fileErr := os.Stat(name)
			if fileErr != nil {
				errors++
				if errors >= maxRetries {
					logger.Errorf("giving up after %d retries: %v", errors, fileErr)
					close(reloadCh)
					return
				} else {
					logger.WithFields(fields).Error(fileErr)
				}
				time.Sleep(retryDelay)
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
			errors = 0
		}
	}()
	return reloadCh
}

func newRestartableCommand(ctx context.Context, cmd string) (command.Cmd, chan<- struct{}) {
	signal := make(chan struct{})
	watchContext, cancelFn := context.WithCancel(ctx)
	go func() {
		<-signal
		cancelFn()
	}()
	return command.NewCommand(watchContext, cmd), signal
}
