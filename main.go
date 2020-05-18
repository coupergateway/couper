package main

import (
	"context"
	"os"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/server"
)

func main() {
	// TODO: command / args
	srv := server.New(command.ContextWithSignal(context.Background()))
	os.Exit(srv.Listen())
}
