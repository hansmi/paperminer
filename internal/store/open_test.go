package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func TestOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")

	s, err := Open(path, 0)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if err := s.Bolt().Sync(); err != nil {
		t.Errorf("Sync() failed: %v", err)
	}

	// Try to open once more.
	if _, err := Open(path, time.Nanosecond); !errors.Is(err, bbolt.ErrTimeout) {
		t.Errorf("Open() didn't fail with timeout: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}
