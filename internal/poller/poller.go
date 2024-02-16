package poller

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/jonboulle/clockwork"
	"go.uber.org/zap"
)

type Options struct {
	Logger *zap.Logger

	// Function invoked to poll (required).
	Poll func(context.Context)

	// Determines the amount of time to sleep before the next poll (required).
	NextDelay func() time.Duration

	// Minimum amount of time to wait between polls. Defaults to 1s.
	MinDelay time.Duration

	// Maximum amount of time to wait between polls.
	MaxDelay time.Duration

	// Amount of random distortion to apply to the calculated delay. Must be in
	// the range [0..+1.0] (0% to 100%).
	Jitter float64

	// Optional channel receiving a message when a new poll is requested.
	Notify <-chan struct{}

	clock clockwork.Clock
}

func (o *Options) nextDelay() time.Duration {
	delay := o.NextDelay()

	if min := o.MinDelay; delay < min {
		delay = min
	} else if max := o.MaxDelay; max != 0 && delay > max {
		delay = max
	}

	if o.Jitter == 0 {
		return delay
	}

	jitter := o.Jitter * (-0.5 + rand.Float64())

	return delay + time.Duration(float64(delay)*jitter)
}

func poll(ctx context.Context, opts Options) error {
	if opts.Poll == nil {
		return fmt.Errorf("%w: Poll is required", os.ErrInvalid)
	}

	if opts.NextDelay == nil {
		return fmt.Errorf("%w: NextDelay is required", os.ErrInvalid)
	}

	if opts.MinDelay == 0 {
		opts.MinDelay = time.Second
	}

	if opts.Jitter < 0 || opts.Jitter > 1.0 {
		return fmt.Errorf("%w: Jitter must be in [0..1.0]", os.ErrInvalid)
	}

	if opts.clock == nil {
		opts.clock = clockwork.NewRealClock()
	}

	var timer clockwork.Timer

	for {
		opts.Poll(ctx)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-opts.Notify:
			continue
		default:
		}

		if delay := opts.nextDelay(); delay > 0 {
			opts.Logger.Info("Waiting for next poll",
				zap.Time("next", opts.clock.Now().Add(delay)),
				zap.Duration("delay", delay),
			)

			if timer == nil {
				timer = opts.clock.NewTimer(delay)
				defer timer.Stop()
			} else {
				timer.Reset(delay)
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("sleep interrupted: %w", ctx.Err())
			case <-timer.Chan():
			case <-opts.Notify:
			}

			if !timer.Stop() {
				select {
				case <-timer.Chan():
				default:
				}
			}
		}
	}
}

// Poll invokes a function repeatedly at a specified interval. The operation
// is stopped when the provided context is canceled.
func Poll(ctx context.Context, opts Options) error {
	err := poll(ctx, opts)

	if errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
		err = nil
	}

	return err
}
