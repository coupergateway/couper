package main

import (
	"context"
	"os"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

func main() {
	// TODO: command / args
	exampleConf := config.Load("example.hcl")

	srv := server.New(command.ContextWithSignal(context.Background()), exampleConf)
	os.Exit(srv.Listen())
}
