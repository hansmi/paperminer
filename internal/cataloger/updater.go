package cataloger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/document"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var errDocumentTooLarge = errors.New("document too large")

type updaterClient interface {
	document.VariantFactsClient

	PatchDocument(context.Context, int64, *plclient.DocumentFields) (*plclient.Document, *plclient.Response, error)
}

type updaterModificationCheckFunc func(context.Context) error

type updaterOptions struct {
	Logger    *zap.Logger
	Resolvers *objectresolver.ObjectResolvers

	Client updaterClient

	Document *plclient.Document
	Metadata *plclient.DocumentMetadata

	TodoTagName   string
	FailedTagName string
	FileSizeMax   int64

	ExtractTimeout   time.Duration
	ExtractFileFacts document.ExtractFileFactsFunc

	CheckModified updaterModificationCheckFunc
}

type updater struct {
	updaterOptions

	todoTag   *plclient.Tag
	failedTag *plclient.Tag
}

func newUpdater(ctx context.Context, opts updaterOptions) (*updater, error) {
	u := &updater{
		updaterOptions: opts,
	}

	for dst, name := range map[**plclient.Tag]string{
		&u.todoTag:   opts.TodoTagName,
		&u.failedTag: opts.FailedTagName,
	} {
		if tag, err := u.Resolvers.Tag.GetOrCreateByName(ctx, name); err != nil {
			return nil, err
		} else {
			*dst = &tag
		}
	}

	return u, nil
}

func (u *updater) checkSize() error {
	var err error

	check := func(name string, size int64) {
		if size > u.FileSizeMax {
			multierr.AppendInto(&err, fmt.Errorf("%s file size of %d is larger than %d bytes",
				name, size, u.FileSizeMax))
		}
	}

	check("original", u.Metadata.OriginalSize)

	if u.Metadata.HasArchiveVersion {
		check("archive", u.Metadata.ArchiveSize)
	}

	if err != nil {
		return fmt.Errorf("%w: %v", errDocumentTooLarge, err)
	}

	return nil
}

func (u *updater) getFacts(ctx context.Context, hasArchiveVersion bool) (*paperminer.Facts, error) {
	var variants []document.Variant

	if hasArchiveVersion {
		variants = append(variants, document.Archived)
	}

	variants = append(variants, document.Original)

	if u.ExtractTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.ExtractTimeout)
		defer cancel()
	}

	return document.ExtractFacts(ctx, document.ExtractFactsOptions{
		Logger:   u.Logger,
		Variants: variants,
		Extract: func(ctx context.Context, v document.Variant) (*paperminer.Facts, error) {
			logger := u.Logger.With(zap.Stringer("document_variant", v))

			return document.ExtractVariantFacts(ctx, document.ExtractVariantFactsOptions{
				Logger:  logger,
				Client:  u.Client,
				Extract: u.ExtractFileFacts,
				ID:      u.Document.ID,
				Variant: v,
			})
		},
	})
}

func (u *updater) patchDocument(ctx context.Context, patch *plclient.DocumentFields) error {
	if len(patch.AsMap()) == 0 {
		return nil
	}

	if err := u.CheckModified(ctx); err != nil {
		return err
	}

	u.Logger.Info("Patching document", zap.Any("patch", patch))

	_, _, err := u.Client.PatchDocument(ctx, u.Document.ID, patch)

	return err
}

func (u *updater) applyFacts(ctx context.Context) error {
	if err := u.checkSize(); err != nil {
		return err
	}

	pb := newPatchBuilder(u.Resolvers, u.Document)

	if facts, err := u.getFacts(ctx, u.Metadata.HasArchiveVersion); err != nil {
		return err
	} else if facts == nil || facts.IsEmpty() {
		u.Logger.Info("No facts found, nothing to do")
	} else {
		u.Logger.Info("Facts found", zap.Any("facts", facts))

		if err := pb.setFacts(ctx, facts); err != nil {
			return err
		}
	}

	pb.unsetTag(u.todoTag.ID)
	pb.unsetTag(u.failedTag.ID)

	return u.patchDocument(ctx, pb.build())
}

func (u *updater) markFailed(ctx context.Context, updateErr error) error {
	u.Logger.Error("Document processing failed permanently", zap.Error(updateErr))

	// TODO: Add note with error to document.
	pb := newPatchBuilder(u.Resolvers, u.Document)
	pb.unsetTag(u.todoTag.ID)
	pb.setTag(u.failedTag.ID)

	return u.patchDocument(ctx, pb.build())
}

// isPermanentError returns whether the error is deemed permanent and not
// retryable.
func isPermanentError(err error) bool {
	var clientReqErr *plclient.RequestError

	return (errors.Is(err, errDocumentTooLarge) ||
		(errors.As(err, &clientReqErr) && clientReqErr.StatusCode == http.StatusNotFound))
}

func (u *updater) Do(ctx context.Context, lastRetry bool) error {
	if err := u.applyFacts(ctx); err != nil {
		if lastRetry || isPermanentError(err) {
			return u.markFailed(ctx, err)
		}

		return err
	}

	return nil
}
