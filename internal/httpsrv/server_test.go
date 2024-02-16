package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestListenAndServe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	readyCh := make(chan net.Addr, 1)

	go func() {
		defer cancel()

		select {
		case addr := <-readyCh:
			if resp, err := http.Get(fmt.Sprintf("http://%s", addr.String())); err != nil {
				t.Errorf("HTTP request failed: %v", err)
			} else if want := http.StatusNotFound; resp.StatusCode != want {
				t.Errorf("Got status code %d, want %d", resp.StatusCode, want)
			}
		}
	}()

	opts := ListenAndServeOptions{
		Logger:          zaptest.NewLogger(t),
		Address:         net.JoinHostPort(netip.IPv6Loopback().String(), "0"),
		Handler:         http.NotFoundHandler(),
		ReadyCh:         readyCh,
		ShutdownTimeout: time.Second,
	}

	if err := ListenAndServe(ctx, opts); err != nil {
		t.Errorf("ListenAndServe() failed: %v", err)
	}
}

func TestListenAndServeListenError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	opts := ListenAndServeOptions{
		Logger:  zaptest.NewLogger(t),
		Network: "<unknown>",
	}

	var unknownNetErr net.UnknownNetworkError

	if err := ListenAndServe(ctx, opts); !errors.As(err, &unknownNetErr) {
		t.Errorf("ListenAndServe() returned unexpected error: %v", err)
	}
}
