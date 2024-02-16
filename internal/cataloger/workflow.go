package cataloger

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/alecthomas/kingpin/v2"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/document"
	"github.com/hansmi/paperminer/internal/facter"
	"github.com/hansmi/paperminer/internal/poller"
	wf "github.com/hansmi/paperminer/internal/workflow"
	"go.uber.org/zap"
)

const minPollInterval = 10 * time.Second
const maxPollInterval = time.Hour

type workflowClient interface {
	walkDocumentsClient
	taskClient
	updaterClient
}

type workflow struct {
	env wf.Environment

	listFacters        bool
	pollInterval       time.Duration
	tagNameTodo        string
	tagNameFailed      string
	fileSizeMax        int64
	retriesMax         int
	factExtractTimeout time.Duration

	facters *facter.Group

	notify chan struct{}
}

func New(ctx context.Context, env wf.Environment) (wf.Workflow, error) {
	w := &workflow{
		env:    env,
		notify: make(chan struct{}, 1),
	}
	w.registerFlags(env.App())

	return w, nil
}

func (w *workflow) registerFlags(app *kingpin.Application) {
	programName := w.env.ProgramName()

	addFlag := func(name, help string) *kingpin.FlagClause {
		return app.Flag("cataloger_"+name, help)
	}

	addFlag("list_facters", "List registered facters and exit.").
		BoolVar(&w.listFacters)

	addFlag("poll_interval",
		fmt.Sprintf("Amount of time to wait between polls for documents (bounded to [%s..%s]).",
			minPollInterval.String(), maxPollInterval.String())).
		Default("1m").
		DurationVar(&w.pollInterval)

	addFlag("tag_todo", "Process documents with this tag. Removed on success.").
		Default(fmt.Sprintf("%s:todo", programName)).
		StringVar(&w.tagNameTodo)

	addFlag("tag_failed", "Tag to apply on permanent failure.").
		Default(fmt.Sprintf("%s:failed", programName)).
		StringVar(&w.tagNameFailed)

	addFlag("retries_max", "Maximum number of retries for processing a document.").
		Default("3").
		IntVar(&w.retriesMax)

	addFlag("fact_extract_timeout", "Maximum amount of time to spend extracting facts from a document.").
		Default("5m").
		DurationVar(&w.factExtractTimeout)

	addFlag("file_size_max_bytes", "Ignore document files exceeding the given amount of bytes.").
		Default(strconv.Itoa(10 * 1024 * 1024)).
		Int64Var(&w.fileSizeMax)
}

func (w *workflow) NotifyPostConsume() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func (w *workflow) processDocumentInner(ctx context.Context, logger *zap.Logger, t *task) error {
	u, err := newUpdater(ctx, updaterOptions{
		Logger:           logger,
		Resolvers:        w.env.Resolvers(),
		TodoTagName:      w.tagNameTodo,
		FailedTagName:    w.tagNameFailed,
		Client:           w.env.Client(),
		Document:         t.doc,
		Metadata:         t.metadata,
		FileSizeMax:      w.fileSizeMax,
		ExtractTimeout:   w.factExtractTimeout,
		ExtractFileFacts: document.MakeFileFactsExtractor(w.facters.Extract),
		CheckModified:    t.CheckModified,
	})
	if err != nil {
		return err
	}

	return u.Do(ctx, t.RetryCount() >= w.retriesMax)
}

func (w *workflow) processDocument(ctx context.Context, logger *zap.Logger, doc *plclient.Document) error {
	opts := taskOptions{
		Logger: logger,
		Store:  w.env.Store(),
		Client: w.env.Client(),
	}

	return processDocument(ctx, doc, opts, w.processDocumentInner,
		func(count int) time.Duration {
			return w.pollInterval * time.Duration(math.Pow(1.5, float64(1+count)))
		},
	)
}

func (w *workflow) processDocuments(ctx context.Context) error {
	tag, err := w.env.Resolvers().Tag.GetOrCreateByName(ctx, w.tagNameTodo)
	if err != nil {
		return err
	}

	return walkDocuments(ctx, w.env.Logger(), w.env.Client(), tag.ID, w.processDocument)
}

func (w *workflow) Validate(ctx context.Context) error {
	facters, err := facter.GroupFromRegistry(w.env.PluginRegistry())
	if err != nil {
		return err
	}

	if w.listFacters {
		for _, i := range facters.Names() {
			fmt.Println(i)
		}

		return wf.ErrValidationEarlyExit
	}

	w.facters = facters

	return nil
}

func (w *workflow) Run(ctx context.Context) error {
	logger := w.env.Logger()

	return poller.Poll(ctx, poller.Options{
		Logger: logger,
		Poll: func(ctx context.Context) {
			if err := w.processDocuments(ctx); err != nil {
				logger.Error("Processing documents failed", zap.Error(err))
			}
		},
		NextDelay: func() time.Duration {
			return w.pollInterval
		},

		MinDelay: minPollInterval,
		MaxDelay: maxPollInterval,
		Jitter:   0.1,

		Notify: w.notify,
	})
}
