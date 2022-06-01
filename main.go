//go:generate go run assets/generate/generate.go
//go:generate go run errors/generate/types.go
//go:generate go fmt errors/types_generated.go

package main

import (
	"context"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/logging/hooks"
	"github.com/avenga/couper/utils"
)

var (
	fields = logrus.Fields{
		"build":   utils.BuildName,
		"version": utils.VersionName,
	}
	testHook logrus.Hook
)

type filesList struct {
	paths []string
}

func main() {
	logrus.Exit(realmain(os.Args))
}

func realmain(arguments []string) int {
	args := command.NewArgs(arguments)
	ctx := context.Background()
	filesList := filesList{}

	type globalFlags struct {
		DebugEndpoint       bool          `env:"debug"`
		FilePath            string        `env:"file"`
		DirPath             string        `env:"file_directory"`
		FileWatch           bool          `env:"watch"`
		FileWatchRetryDelay time.Duration `env:"watch_retry_delay"`
		FileWatchRetries    int           `env:"watch_retries"`
		LogFormat           string        `env:"log_format"`
		LogLevel            string        `env:"log_level"`
		LogPretty           bool          `env:"log_pretty"`
	}
	var flags globalFlags

	set := flag.NewFlagSet("global options", flag.ContinueOnError)
	set.BoolVar(&flags.DebugEndpoint, "debug", false, "-debug")
	set.Var(&filesList, "f", "-f /path/to/couper.hcl ...")
	set.Var(&filesList, "d", "-d /path/to/couper.d/ ...")
	set.BoolVar(&flags.FileWatch, "watch", false, "-watch")
	set.DurationVar(&flags.FileWatchRetryDelay, "watch-retry-delay", time.Millisecond*500, "-watch-retry-delay 1s")
	set.IntVar(&flags.FileWatchRetries, "watch-retries", 5, "-watch-retries 10")
	set.StringVar(&flags.LogFormat, "log-format", config.DefaultSettings.LogFormat, "-log-format=json")
	set.StringVar(&flags.LogLevel, "log-level", config.DefaultSettings.LogLevel, "-log-level info")
	set.BoolVar(&flags.LogPretty, "log-pretty", config.DefaultSettings.LogPretty, "-log-pretty")

	if len(args) == 0 || command.NewCommand(ctx, args[0]) == nil {
		command.Synopsis()
		set.Usage()
		return 1
	}

	cmd := args[0]
	args = args[1:]

	if cmd != "run" && cmd != "verify" { // global options are not required atm, fast exit.
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
		newLogger(flags.LogFormat, flags.LogLevel, flags.LogPretty).Error(err)
		return 1
	}
	env.Decode(&flags)

	if len(filesList.paths) == 0 {
		// Get paths from COUPER_FILE and then COUPER_DIRECTORY
		if flags.FilePath != "" {
			filesList.paths = append(filesList.paths, flags.FilePath)
		}

		if flags.DirPath != "" {
			filesList.paths = append(filesList.paths, flags.DirPath)
		}

		if len(filesList.paths) == 0 {
			filesList.paths = append(filesList.paths, config.DefaultFilename)
		}
	}

	if cmd == "verify" {
		log := newLogger(flags.LogFormat, flags.LogLevel, flags.LogPretty)

		err = command.NewCommand(ctx, cmd).Execute(filesList.paths, nil, log)
		if err != nil {
			return 1
		}
		return 0
	}

	confFile, err := configload.LoadFiles(filesList.paths)
	if err != nil {
		newLogger(flags.LogFormat, flags.LogLevel, flags.LogPretty).WithError(err).Error()
		return 1
	}

	// The file gets initialized with the default settings, flag args are preferred over file settings.
	// Only override file settings if the flag value differ from the default.
	if flags.LogFormat != config.DefaultSettings.LogFormat {
		confFile.Settings.LogFormat = flags.LogFormat
	}
	if flags.LogLevel != config.DefaultSettings.LogLevel {
		confFile.Settings.LogLevel = flags.LogLevel
	}
	if flags.LogPretty != config.DefaultSettings.LogPretty {
		confFile.Settings.LogPretty = flags.LogPretty
	}
	logger := newLogger(confFile.Settings.LogFormat, confFile.Settings.LogLevel, confFile.Settings.LogPretty)

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
	}).Info("watching configuration file(s)")
	errCh := make(chan error, 1)
	errRetries := 0

	execCmd, restartSignal := newRestartableCommand(ctx, cmd)
	go func() {
		errCh <- execCmd.Execute(args, confFile, logger)
	}()

	if flags.DebugEndpoint {
		debugListenAndServe(logger)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	reloadCh := watchConfigFiles(confFile.Files, logger, flags.FileWatchRetries, flags.FileWatchRetryDelay)
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
		case <-sigCh:
			close(restartSignal)
			return 0
		case _, more := <-reloadCh:
			if !more {
				return 1
			}
			errRetries = 0 // reset
			logger.Info("reloading couper configuration")

			cf, reloadErr := configload.LoadFiles(filesList.paths)
			if reloadErr != nil {
				logger.WithError(reloadErr).Error("reload failed")
				time.Sleep(flags.FileWatchRetryDelay)
				continue
			}
			// dry run configuration
			tmpStoreCh := make(chan struct{})
			tmpMemStore := cache.New(logger, tmpStoreCh)
			_, reloadErr = runtime.NewServerConfiguration(cf, logger.WithFields(fields), tmpMemStore)
			close(tmpStoreCh)
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
func newLogger(format, level string, pretty bool) *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout
	parsedLevel, err := logrus.ParseLevel(level)
	if err != nil {
		parsedLevel = logrus.InfoLevel
		logger.Error("parsing log level, falling back to info level")
	}
	logger.Level = parsedLevel

	logger.AddHook(&hooks.Error{})
	logger.AddHook(&hooks.Context{})
	logger.AddHook(&hooks.CustomLogs{})

	if testHook != nil {
		logger.AddHook(testHook)
		logger.Out = io.Discard
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
		logger.SetFormatter(hooks.NewJSONColorFormatter(logConf.ParentFieldKey, settings.LogPretty))
	}
	return logger.WithField("type", logConf.TypeFieldKey).WithFields(fields)
}

func getWatchFilesList(watchFiles []string) (map[string]struct{}, error) {
	files := make(map[string]struct{})

	for _, file := range watchFiles {
		files[file] = struct{}{}
	}

	return files, nil
}

func retryWatching(errorsSeen, maxRetries int, retryDelay time.Duration,
	logger logrus.FieldLogger, err error, reloadCh chan struct{}) bool {

	if errorsSeen >= maxRetries {
		logger.Errorf("giving up after %d retries: %v", errorsSeen, err)
		close(reloadCh)

		return false
	}

	logger.WithFields(fields).Error(err)

	time.Sleep(retryDelay)

	return true
}

func syncWatchFilesList(new map[string]struct{}, old map[string]time.Time) map[string]time.Time {
	synced := make(map[string]time.Time)

	for name := range new {
		if t, ok := old[name]; ok {
			synced[name] = t
		} else {
			synced[name] = time.Time{}
		}
	}

	return synced
}

func watchConfigFiles(watchFiles []string, logger logrus.FieldLogger, maxRetries int, retryDelay time.Duration) <-chan struct{} {
	reloadCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second / 4)
		defer ticker.Stop()
		var errorsSeen int

		lastChanges := make(map[string]time.Time)

		files, err := getWatchFilesList(watchFiles)
		if err != nil {
			logger.Error(err)
			close(reloadCh)
			return
		}

		for name := range files {
			lastChanges[name] = time.Time{}
		}

	watchFiles:
		for {
			<-ticker.C

			// Compare files list
			filesList, err := getWatchFilesList(watchFiles)
			if err != nil {
				errorsSeen++

				if !retryWatching(errorsSeen, maxRetries, retryDelay, logger, err, reloadCh) {
					return
				}

				continue watchFiles
			}

			if len(filesList) != len(lastChanges) {
				lastChanges = syncWatchFilesList(filesList, lastChanges)

				errorsSeen = 0
				reloadCh <- struct{}{}

				continue watchFiles
			}

			for f, t := range lastChanges {
				info, err := os.Stat(f)

				if err != nil {
					errorsSeen++

					if !retryWatching(errorsSeen, maxRetries, retryDelay, logger, err, reloadCh) {
						return
					}

					continue watchFiles
				}

				if t.IsZero() { // first round
					lastChanges[f] = info.ModTime()
					continue watchFiles
				}

				if info.ModTime().After(t) {
					reloadCh <- struct{}{}
				}

				lastChanges[f] = info.ModTime()

				errorsSeen = 0
			}
		}
	}()

	return reloadCh
}

func newRestartableCommand(ctx context.Context, cmd string) (command.Cmd, chan<- struct{}) {
	sig := make(chan struct{}, 1)
	watchContext, cancelFn := context.WithCancel(ctx)
	go func() {
		<-sig
		cancelFn()
	}()
	return command.NewCommand(watchContext, cmd), sig
}

func debugListenAndServe(logEntry *logrus.Entry) {
	const tracePort = "6060"
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	traceSrv := http.Server{Addr: ":" + tracePort}
	traceSrv.Handler = mux
	go func() {
		logEntry.WithField("debug", "pprof").WithField("port", tracePort).Info("listening")
		if e := traceSrv.ListenAndServe(); e != nil {
			logEntry.WithField("debug", "pprof").Error(e)
		}
	}()
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()
		_ = traceSrv.Shutdown(ctx)
	}()
}

func (list *filesList) String() string {
	return ""
}

func (list *filesList) Set(value string) error {
	list.paths = append(list.paths, value)
	return nil
}
