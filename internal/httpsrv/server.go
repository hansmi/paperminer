package httpsrv

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type ListenAndServeOptions struct {
	Logger *zap.Logger

	Network string
	Address string

	Handler http.Handler

	ReadyCh chan<- net.Addr

	ShutdownTimeout time.Duration
}

// Listen on a TCP port and serve HTTP requests. The HTTP server is stopped
// when the provided context is canceled.
func ListenAndServe(ctx context.Context, opts ListenAndServeOptions) error {
	if opts.Network == "" {
		opts.Network = "tcp"
	}

	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = 10 * time.Second
	}

	listener, err := net.Listen(opts.Network, opts.Address)
	if err != nil {
		return err
	}

	defer listener.Close()

	logger := opts.Logger.With(zap.String("addr", listener.Addr().String()))
	logger.Info("HTTP server listening")

	srv := http.Server{
		Handler: opts.Handler,
	}

	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.ShutdownTimeout)
			defer cancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				logger.Error("HTTP server shutdown failed", zap.Error(err))
			}
		}
	}()

	if opts.ReadyCh != nil {
		opts.ReadyCh <- listener.Addr()
		close(opts.ReadyCh)
	}

	if err := srv.Serve(listener); err != nil && !(errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil) {
		return err
	}

	<-shutdownDone

	return nil
}
