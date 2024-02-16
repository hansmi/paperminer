package store

import (
	"fmt"
	"time"

	"github.com/timshannon/bolthold"
	"go.etcd.io/bbolt"
)

func Open(path string, timeout time.Duration) (*bolthold.Store, error) {
	var opts bolthold.Options

	opts.Options = &bbolt.Options{
		Timeout: timeout,
	}

	db, err := bolthold.Open(path, 0o600, &opts)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}

	if err := db.ReIndex(&DocumentTask{}, nil); err != nil {
		return nil, fmt.Errorf("store indexing: %w", err)
	}

	return db, nil
}
