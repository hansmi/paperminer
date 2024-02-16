package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestPrune(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	s, err := Open(filepath.Join(t.TempDir(), "file"), 0)
	if err != nil {
		t.Errorf("Open() failed: %v", err)
	}

	logger := zaptest.NewLogger(t)

	if err := Prune(ctx, logger, s, time.Now()); err != nil {
		t.Errorf("Prune(now) on empty store failed: %v", err)
	}

	start := time.Date(2023, time.January, 1, 1, 0, 0, 0, time.UTC)
	count := 100

	for i := 0; i < count; i++ {
		if err := s.Insert(i, DocumentTask{
			RecordUpdated: start.Add(time.Duration(i) * time.Hour),
		}); err != nil {
			t.Errorf("Insert(%d) failed: %v", i, err)
		}
	}

	if err := Prune(ctx, logger, s, start.Add(33*time.Hour)); err != nil {
		t.Errorf("Prune() failed: %v", err)
	}

	if got, err := s.Count(DocumentTask{}, nil); err != nil {
		t.Errorf("Count() failed: %v", err)
	} else if want := count - 33; got != want {
		t.Errorf("Counted %d records after Prune(), want %d", got, want)
	}
}
