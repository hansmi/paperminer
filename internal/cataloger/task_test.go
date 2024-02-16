package cataloger

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/ref"
	"github.com/hansmi/paperminer/internal/store"
	"github.com/jonboulle/clockwork"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type fakeTaskStore struct {
	t        *testing.T
	keyCache []byte
	rec      any
}

func (s *fakeTaskStore) verifyKey(key any) {
	if s.keyCache == nil {
		s.keyCache = bytes.Clone(key.([]byte))
	} else {
		if diff := cmp.Diff(s.keyCache, key); diff != "" {
			s.t.Errorf("Key diff (-want +got):\n%s", diff)
		}
	}
}

func (s *fakeTaskStore) Get(key any, rec any) error {
	s.verifyKey(key)

	if s.rec == nil {
		return bolthold.ErrNotFound
	}

	r := reflect.Indirect(reflect.ValueOf(rec))
	r.Set(reflect.ValueOf(s.rec))

	return nil
}

func (s *fakeTaskStore) Upsert(key any, rec any) error {
	s.verifyKey(key)

	s.rec = rec

	return nil
}

type fakeTaskClient struct {
	doc      plclient.Document
	metadata plclient.DocumentMetadata
}

func (c *fakeTaskClient) GetDocument(context.Context, int64) (*plclient.Document, *plclient.Response, error) {
	return &c.doc, nil, nil
}

func (c *fakeTaskClient) GetDocumentMetadata(context.Context, int64) (*plclient.DocumentMetadata, *plclient.Response, error) {
	return &c.metadata, nil, nil
}

func TestLoadTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	s := &fakeTaskStore{t: t}
	client := &fakeTaskClient{}

	clock := clockwork.NewFakeClockAt(time.Unix(1234567890, 0))

	doc := &plclient.Document{}

	taskOpts := taskOptions{
		Logger: zaptest.NewLogger(t),
		Store:  s,
		Client: client,
		clock:  clock,
	}

	task, err := loadTask(ctx, doc, taskOpts)
	if err != nil {
		t.Fatalf("loadTask() failed: %v", err)
	}

	if got := task.RetryCount(); got != 0 {
		t.Errorf("RetryCount() = %d, want zero", got)
	}

	if err := task.CheckModified(ctx); err != nil {
		t.Errorf("CheckModified() failed: %v", err)
	}

	clock.Advance(time.Minute)

	if err := task.SaveResult(nil, 0); err != nil {
		t.Errorf("SaveResult(nil) failed: %v", err)
	}

	if task, err = loadTask(ctx, doc, taskOpts); err != nil {
		t.Fatalf("loadTask() failed: %v", err)
	}

	if diff := cmp.Diff(store.DocumentTask{
		RecordCreated: time.Unix(1234567890, 0),
		RecordUpdated: time.Unix(1234567890+60, 0),
		Attempts: []store.DocumentTaskAttempt{
			{
				Begin:   time.Unix(1234567890, 0),
				End:     time.Unix(1234567890+60, 0),
				Success: true,
			},
		},
	}, s.rec,
		cmpopts.EquateEmpty(),
	); diff != "" {
		t.Errorf("Record diff (-want +got):\n%s", diff)
	}

	clock.Advance(time.Minute)

	if err := task.SaveResult(errors.New("test error"), 11*time.Second); err != nil {
		t.Errorf("SaveResult(non-nil) failed: %v", err)
	}

	if diff := cmp.Diff(store.DocumentTask{
		RecordCreated: time.Unix(1234567890, 0),
		RecordUpdated: time.Unix(1234567890+120, 0),
		RetryCount:    1,
		RetryAfter:    time.Unix(1234567890+120+11, 0),
		Attempts: []store.DocumentTaskAttempt{
			{
				Begin:   time.Unix(1234567890, 0),
				End:     time.Unix(1234567890+60, 0),
				Success: true,
			},
			{
				Begin:   time.Unix(1234567890+60, 0),
				End:     time.Unix(1234567890+120, 0),
				Message: "test error",
			},
		},
	}, s.rec,
		cmpopts.EquateEmpty(),
	); diff != "" {
		t.Errorf("Record diff (-want +got):\n%s", diff)
	}

	if task, err = loadTask(ctx, doc, taskOpts); err != nil {
		t.Fatalf("loadTask() failed: %v", err)
	} else if task != nil {
		t.Errorf("loadTask() returned non-nil value even though retry time hasn't passed yet: %#v", task)
	}
}

func TestTaskCheckModified(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mod     func(*fakeTaskClient)
		wantErr error
	}{
		{
			name: "defaults",
		},
		{
			name: "doc only",
			mod: func(c *fakeTaskClient) {
				c.doc.Title = "changed"
			},
			wantErr: errConcurrentModification,
		},
		{
			name: "metadata only",
			mod: func(c *fakeTaskClient) {
				c.metadata.ArchiveChecksum = "1234"
			},
			wantErr: errConcurrentModification,
		},
		{
			name: "doc and metadata",
			mod: func(c *fakeTaskClient) {
				c.doc.Owner = ref.Ref[int64](100)
				c.metadata.HasArchiveVersion = true
			},
			wantErr: errConcurrentModification,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			s := &fakeTaskStore{t: t}
			client := &fakeTaskClient{}

			clock := clockwork.NewFakeClockAt(time.Unix(987654321, 0))

			doc := &plclient.Document{}

			taskOpts := taskOptions{
				Logger: zaptest.NewLogger(t),
				Store:  s,
				Client: client,
				clock:  clock,
			}

			task, err := loadTask(ctx, doc, taskOpts)
			if err != nil {
				t.Fatalf("loadTask() failed: %v", err)
			}

			if err := task.CheckModified(ctx); err != nil {
				t.Errorf("CheckModified() failed: %v", err)
			}

			if tc.mod != nil {
				tc.mod(client)
			}

			err = task.CheckModified(ctx)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessDocument(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	s, err := store.Open(filepath.Join(t.TempDir(), "data"), 0)
	if err != nil {
		t.Errorf("Opening store failed: %v", err)
	}

	for _, tc := range []struct {
		name       string
		processErr error
	}{
		{
			name: "success",
		},
		{
			name:       "error",
			processErr: errors.New("test error"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeTaskClient{}

			clock := clockwork.NewFakeClockAt(time.Unix(1234567890, 0))

			opts := taskOptions{
				Logger: zaptest.NewLogger(t),
				Store:  s,
				Client: client,
				clock:  clock,
			}

			err = processDocument(ctx, &client.doc, opts,
				func(ctx context.Context, _ *zap.Logger, task *task) error {
					if err := task.CheckModified(ctx); err != nil {
						t.Errorf("CheckModified() failed: %v", err)
					}

					return tc.processErr
				},
				func(count int) time.Duration {
					return time.Duration(count) * time.Minute
				},
			)

			if err != nil {
				t.Errorf("processDocument() failed: %v", err)
			}
		})
	}
}
