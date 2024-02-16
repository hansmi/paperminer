package core

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/hansmi/paperminer/internal/store"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type fakeStorePrunerEnv struct {
	logger *zap.Logger
	store  *bolthold.Store
}

func (e *fakeStorePrunerEnv) Logger() *zap.Logger {
	return e.logger
}

func (e *fakeStorePrunerEnv) Store() *bolthold.Store {
	return e.store
}

func TestStorePruner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	wf, err := newStorePruner(ctx, nil)
	if err != nil {
		t.Errorf("newStorePruner() failed: %v", err)
	}

	s, err := store.Open(filepath.Join(t.TempDir(), "db"), 0)
	if err != nil {
		t.Errorf("Open() failed: %v", err)
	}

	count := 0

	env := &fakeStorePrunerEnv{
		logger: zaptest.NewLogger(t),
		store:  s,
	}

	p := wf.(*storePruner)
	p.env = env
	p.minDelay = time.Nanosecond
	p.nextDelay = func() time.Duration {
		if count++; count > 3 {
			cancel()
		}

		return 0
	}

	if err := p.Run(ctx); err != nil {
		t.Errorf("Run() failed: %v", err)
	}
}
