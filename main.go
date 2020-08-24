//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

var (
	configFile = flag.String("f", "couper.hcl", "-f ./couper.conf")
	listenPort = flag.Int("p", server.DefaultHTTPConfig.ListenPort, "-p 8080")
	useXFH     = flag.Bool("xfh", false, "-xfh")
)

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}

	logger := newLogger()

	if *listenPort < 0 || *listenPort > 65535 {
		logger.Fatalf("Invalid listen port given: %d", *listenPort)
	}

	configuration, err := config.LoadFile(*configFile)
	if err != nil {
		logger.Fatal(err)
	}

	configuration.ListenPort = *listenPort
	if p := os.Getenv("COUPER_PORT"); p != "" {
		port, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			logger.Fatal(err)
		}
		if port < 0 || port > 65535 {
			logger.Fatalf("Invalid COUPER_PORT given: %d", port)
		}

		configuration.ListenPort = int(port)
	}

	configuration.UseXFH = *useXFH
	switch os.Getenv("COUPER_XFH") {
	case "true":
		configuration.UseXFH = true
	case "false":
		configuration.UseXFH = false
	}

	err = os.Chdir(filepath.Dir(*configFile))
	if err != nil {
		logger.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}
	configuration.WorkDir = wd

	ctx := command.ContextWithSignal(context.Background())
	srv := server.New(ctx, logger, configuration)
	srv.Listen()
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
