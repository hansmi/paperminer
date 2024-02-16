package cataloger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/store"
	"github.com/jonboulle/clockwork"
	jd "github.com/josephburnett/jd/lib"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
)

var errConcurrentModification = errors.New("detected concurrent modification")

func convertToJsonNode(data any) (jd.JsonNode, error) {
	buf, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling %#v to JSON: %w", data, err)
	}

	return jd.ReadJsonString(string(buf))
}

type docSnapshot struct {
	ID       int64
	Added    time.Time
	Created  time.Time
	Modified time.Time
	Title    string
	Content  string

	OriginalChecksum  string
	HasArchiveVersion bool
	ArchiveChecksum   string
}

func newDocSnapshot(doc *plclient.Document, metadata *plclient.DocumentMetadata) docSnapshot {
	return docSnapshot{
		ID:       doc.ID,
		Added:    doc.Added,
		Created:  doc.Created,
		Modified: doc.Modified,
		Title:    doc.Title,
		Content:  doc.Content,

		OriginalChecksum:  metadata.OriginalChecksum,
		HasArchiveVersion: metadata.HasArchiveVersion,
		ArchiveChecksum:   metadata.ArchiveChecksum,
	}
}

type taskStore interface {
	Get(any, any) error
	Upsert(any, any) error
}

type taskClient interface {
	GetDocument(context.Context, int64) (*plclient.Document, *plclient.Response, error)
	GetDocumentMetadata(context.Context, int64) (*plclient.DocumentMetadata, *plclient.Response, error)
}

type taskOptions struct {
	Logger *zap.Logger
	Store  taskStore
	Client taskClient

	clock clockwork.Clock
}

type task struct {
	opts taskOptions

	doc      *plclient.Document
	metadata *plclient.DocumentMetadata

	snapshot docSnapshot

	begin time.Time

	key []byte
	rec store.DocumentTask
}

func loadTask(ctx context.Context, doc *plclient.Document, opts taskOptions) (*task, error) {
	if opts.clock == nil {
		opts.clock = clockwork.NewRealClock()
	}

	metadata, _, err := opts.Client.GetDocumentMetadata(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("getting document %d metadata: %w", doc.ID, err)
	}

	opts.Logger.Debug("Document information",
		zap.Any("document", doc),
		zap.Any("metadata", metadata),
	)

	t := &task{
		opts: opts,

		doc:      doc,
		metadata: metadata,

		begin: opts.clock.Now(),

		rec: store.DocumentTask{
			ID: doc.ID,

			Added:    doc.Added,
			Modified: doc.Modified,

			OriginalChecksum: metadata.OriginalChecksum,
			ArchiveChecksum:  metadata.ArchiveChecksum,
		},
	}

	if t.key, err = t.rec.Key(); err != nil {
		return nil, err
	}

	if err := opts.Store.Get(t.key, &t.rec); errors.Is(err, bolthold.ErrNotFound) {
		t.rec.RecordCreated = t.begin
	} else if err != nil {
		return nil, fmt.Errorf("getting record for %q: %w", t.key, err)
	} else if retryAfter := t.rec.RetryAfter; !retryAfter.IsZero() && retryAfter.After(t.begin) {
		opts.Logger.Info("Document failed previously and retry time has not been reached yet",
			zap.Time("retry_after", retryAfter),
			zap.Any("attempts", t.rec.Attempts))
		return nil, nil
	}

	t.snapshot = newDocSnapshot(doc, metadata)

	return t, nil
}

func (t *task) RetryCount() int {
	return t.rec.RetryCount
}

func (t *task) CheckModified(ctx context.Context) error {
	curDoc, _, err := t.opts.Client.GetDocument(ctx, t.doc.ID)
	if err != nil {
		return fmt.Errorf("getting document %d for conflict check: %w", t.doc.ID, err)
	}

	curMetadata, _, err := t.opts.Client.GetDocumentMetadata(ctx, t.doc.ID)
	if err != nil {
		return fmt.Errorf("getting document %d metadata for conflict check: %w", t.doc.ID, err)
	}

	originalSnapshotNode, err := convertToJsonNode(t.snapshot)
	if err != nil {
		return err
	}

	curSnapshot := newDocSnapshot(curDoc, curMetadata)

	curSnapshotNode, err := convertToJsonNode(curSnapshot)
	if err != nil {
		return err
	}

	if diff := originalSnapshotNode.Diff(curSnapshotNode); len(diff) > 0 {
		return fmt.Errorf("%w:\n%s", errConcurrentModification, diff.Render())
	}

	return nil
}

func (t *task) SaveResult(err error, retryAfter time.Duration) error {
	now := t.opts.clock.Now()
	rec := &t.rec

	attempt := store.DocumentTaskAttempt{
		Success: (err == nil),
		Begin:   t.begin,
		End:     now,
	}

	if !attempt.Success {
		rec.RetryCount++
		rec.RetryAfter = now.Add(retryAfter)
		attempt.Message = err.Error()
	}

	rec.RecordUpdated = now
	rec.Attempts = append(rec.Attempts, attempt)

	if err := t.opts.Store.Upsert(t.key, *rec); err != nil {
		return fmt.Errorf("inserting/updating record for %q: %w", t.key, err)
	}

	return nil
}

func processDocument(ctx context.Context,
	doc *plclient.Document,
	opts taskOptions,
	fn func(context.Context, *zap.Logger, *task) error,
	calcRetryDelay func(int) time.Duration,
) error {
	task, err := loadTask(ctx, doc, opts)
	if err != nil {
		return err
	} else if task == nil {
		// task not yet ready
		return nil
	}

	processErr := fn(ctx, opts.Logger, task)

	retryDelay := calcRetryDelay(1 + task.RetryCount())

	if processErr != nil {
		opts.Logger.Error("Processing document failed",
			zap.Error(processErr),
			zap.Duration("retry_delay", retryDelay),
		)
	}

	if err := task.SaveResult(processErr, retryDelay); err != nil {
		return fmt.Errorf("saving processing result: %w", err)
	}

	return nil
}
