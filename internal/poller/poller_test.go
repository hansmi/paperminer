package poller

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jonboulle/clockwork"
	"go.uber.org/zap/zaptest"
)

func TestBadOption(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(*Options)
	}{
		{
			name: "poll",
			apply: func(o *Options) {
				o.Poll = nil
			},
		},
		{
			name: "jitter",
			apply: func(o *Options) {
				o.Jitter = 100
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			opts := Options{
				Logger: zaptest.NewLogger(t),
				Poll: func(context.Context) {
				},
				NextDelay: func() time.Duration {
					return time.Nanosecond
				},
			}

			if err := Poll(ctx, opts); err != nil {
				t.Errorf("Poll() failed: %v", err)
			}

			tc.apply(&opts)

			err := Poll(ctx, opts)

			if diff := cmp.Diff(os.ErrInvalid, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNextDelay(t *testing.T) {
	for _, tc := range []struct {
		name         string
		opts         Options
		wantMinDelay time.Duration
		wantMaxDelay time.Duration
	}{
		{
			name: "defaults",
			opts: Options{
				NextDelay: func() time.Duration {
					return time.Millisecond
				},
			},
			wantMinDelay: time.Millisecond,
			wantMaxDelay: time.Millisecond,
		},
		{
			name: "jitter",
			opts: Options{
				NextDelay: func() time.Duration {
					return time.Millisecond
				},
				MinDelay: time.Nanosecond,
				MaxDelay: time.Second,
				Jitter:   0.2,
			},
			wantMinDelay: 850 * time.Microsecond,
			wantMaxDelay: 1150 * time.Microsecond,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.opts.clock = clockwork.NewFakeClock()

			for i := 0; i < 100; i++ {
				if delay := tc.opts.nextDelay(); delay < 0 {
					t.Errorf("nextDelay() returned negative value: %v", delay)
				} else if delay < tc.wantMinDelay {
					t.Errorf("nextDelay() = %v, want more than %v", delay, tc.wantMinDelay)
				} else if delay > tc.wantMaxDelay {
					t.Errorf("nextDelay() = %v, want less than %v", delay, tc.wantMaxDelay)
				}
			}
		})
	}
}

func TestPoll(t *testing.T) {
	calls := 0

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	fc := clockwork.NewFakeClock()

	opts := Options{
		Logger: zaptest.NewLogger(t),
		Poll: func(context.Context) {
			go func() {
				fc.BlockUntil(1)
				fc.Advance(time.Millisecond)
			}()

			if calls == 100 {
				cancel()
			}

			calls++
		},
		NextDelay: func() time.Duration {
			return time.Nanosecond
		},
		MinDelay: time.Nanosecond,
		MaxDelay: time.Millisecond,

		clock: fc,
	}

	if err := Poll(ctx, opts); err != nil {
		t.Errorf("Poll() failed: %v", err)
	}

	if want := 100; calls < want {
		t.Errorf("Too few callbacks: %d < %d", calls, want)
	}
}

func TestRunCancelInSleep(t *testing.T) {
	fc := clockwork.NewFakeClock()

	opts := Options{
		Logger: zaptest.NewLogger(t),
		Poll: func(context.Context) {
		},
		NextDelay: func() time.Duration {
			return time.Minute
		},

		clock: fc,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		fc.BlockUntil(1)
		cancel()
	}()

	if err := Poll(ctx, opts); err != nil {
		t.Errorf("Poll() failed: %v", err)
	}
}

func TestNotify(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	fc := clockwork.NewFakeClock()
	ch := make(chan struct{}, 10)
	calls := 0

	opts := Options{
		Logger: zaptest.NewLogger(t),
		Poll: func(context.Context) {
			if calls++; calls < 3 {
				ch <- struct{}{}
			} else {
				cancel()
			}
		},
		NextDelay: func() time.Duration {
			return time.Minute
		},
		Notify: ch,

		clock: fc,
	}

	go func() {
		fc.BlockUntil(1)
		ch <- struct{}{}
	}()

	if err := Poll(ctx, opts); err != nil {
		t.Errorf("Poll() failed: %v", err)
	}

	if want := 3; calls < want {
		t.Errorf("Got %d polls, want at least %d", calls, want)
	}
}
