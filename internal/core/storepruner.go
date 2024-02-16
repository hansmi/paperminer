package core

import (
	"context"
	"time"

	"github.com/hansmi/paperminer/internal/poller"
	"github.com/hansmi/paperminer/internal/store"
	"github.com/hansmi/paperminer/internal/workflow"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
)

type storePrunerEnv interface {
	Logger() *zap.Logger
	Store() *bolthold.Store
}

type storePruner struct {
	env       storePrunerEnv
	minDelay  time.Duration
	nextDelay func() time.Duration
}

func newStorePruner(_ context.Context, env workflow.Environment) (workflow.Workflow, error) {
	return &storePruner{
		env: env,
		nextDelay: func() time.Duration {
			return time.Hour
		},
	}, nil
}

func (p *storePruner) Validate(ctx context.Context) error {
	return nil
}

func (p *storePruner) Run(ctx context.Context) error {
	const deleteUpdatedAfter = 24 * time.Hour

	logger := p.env.Logger()

	return poller.Poll(ctx, poller.Options{
		Logger: logger,
		Poll: func(ctx context.Context) {
			deleteUpdatedBefore := time.Now().Add(-deleteUpdatedAfter)

			if err := store.Prune(ctx, logger, p.env.Store(), deleteUpdatedBefore); err != nil {
				logger.Error("Store pruning failed", zap.Error(err))
			}
		},
		MinDelay:  p.minDelay,
		NextDelay: p.nextDelay,
		Jitter:    0.1,
	})
}
