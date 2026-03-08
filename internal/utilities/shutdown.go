package utilities

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

func Wait(ctx context.Context, timeout time.Duration, hooks ...func(context.Context) error) error {
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-sigCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var g errgroup.Group
	for _, h := range hooks {
		g.Go(func() error {
			return h(shutdownCtx)
		})
	}

	return g.Wait()
}
