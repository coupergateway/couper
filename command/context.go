package command

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ContextWithSignal creates a context canceled when SIGINT or SIGTERM are notified
func ContextWithSignal(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		cancel()
	}()
	return ctx
}
