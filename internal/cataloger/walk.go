package cataloger

import (
	"context"
	"runtime"

	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type walkDocumentsClient interface {
	ListAllDocuments(context.Context, plclient.ListDocumentsOptions, func(context.Context, plclient.Document) error) error
}

type walkDocumentsHandler func(context.Context, *zap.Logger, *plclient.Document) error

func walkDocuments(ctx context.Context, logger *zap.Logger, cl walkDocumentsClient, tagID int64, process walkDocumentsHandler) error {
	var opts plclient.ListDocumentsOptions

	opts.Ordering.Field = "id"
	opts.Ordering.Desc = false
	opts.Tags.ID = &tagID

	tasks := pool.New().WithMaxGoroutines(runtime.GOMAXPROCS(0))

	defer tasks.Wait()

	for seen := map[int64]struct{}{}; ; {
		var found bool

		if err := cl.ListAllDocuments(ctx, opts, func(_ context.Context, doc plclient.Document) error {
			// Process each document only once
			if _, ok := seen[doc.ID]; ok {
				return nil
			}

			seen[doc.ID] = struct{}{}
			found = true

			tasks.Go(func() {
				logger := logger.With(zap.Int64("document_id", doc.ID))
				logger.Info("Document info",
					zap.Time("added", doc.Added),
					zap.String("original_filename", doc.OriginalFileName),
					zap.Stringp("archived_filename", doc.ArchivedFileName))

				if err := process(ctx, logger, &doc); err != nil {
					logger.Error("Error while processing document", zap.Error(err))
				}
			})

			return nil
		}); err != nil {
			return err
		}

		// Processed documents may shift others on later result pages and new
		// documents may have been added in the meantime. Repeat the process
		// until no new documents are found.
		if !found {
			break
		}
	}

	return nil
}
