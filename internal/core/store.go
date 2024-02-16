package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hansmi/paperminer/internal/store"
	"github.com/timshannon/bolthold"
	"go.uber.org/multierr"
)

// openDefaultStore creates a new store in a temporary location. The returned
// cleanup function should be called when the store is closed and no longer
// used (usually on process termination).
func openDefaultStore(dir string) (_ *bolthold.Store, cleanup func() error, err error) {
	if dir == "" {
		if dir, err = os.UserCacheDir(); err != nil {
			return nil, nil, fmt.Errorf("user cache dir: %w", err)
		}
	}

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0o0700); err != nil {
			return nil, nil, fmt.Errorf("creating store directory: %w", err)
		}
	}

	tmpdir, err := os.MkdirTemp(dir, "paperminer-*")
	if err != nil {
		return nil, nil, err
	}

	defer multierr.AppendFunc(&err, func() error {
		return os.RemoveAll(tmpdir)
	})

	s, err := store.Open(filepath.Join(tmpdir, "store.bin"), 0)
	if err != nil {
		return nil, nil, err
	}

	return s, func() error {
		// No-op on POSIX systems. Removal of the temporary directory and its
		// contents may need to be delayed until cleanup on Windows.
		return nil
	}, nil
}
