package document

import (
	"context"
	"fmt"
	"io"
	"os"

	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/facter"
	"github.com/hansmi/paperminer/internal/fsutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type VariantFactsClient interface {
	DownloadDocumentOriginal(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error)
	DownloadDocumentArchived(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error)
}

type docDownloadFunc func(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error)

type ExtractFileFactsFunc func(context.Context, *zap.Logger, string) (facter.FactsSlice, error)

type ExtractVariantFactsOptions struct {
	Logger *zap.Logger

	// Directory for storing temporary files. May be empty to use the system
	// default.
	Basedir string

	Client VariantFactsClient

	// Function extracting facts from a document file.
	Extract ExtractFileFactsFunc

	ID      int64
	Variant Variant
}

func selectDownloadFunction(cl VariantFactsClient, v Variant) (docDownloadFunc, error) {
	table := map[Variant]docDownloadFunc{
		Original: cl.DownloadDocumentOriginal,
		Archived: cl.DownloadDocumentArchived,
	}

	if fn, ok := table[v]; ok {
		return fn, nil
	}

	return nil, fmt.Errorf("missing download function for variant %q", v.String())
}

// Download a document into a temporary file. The caller is responsible for
// removing the directory when the document is no longer used.
func download(ctx context.Context, logger *zap.Logger, tmpdir string, fn docDownloadFunc, id int64) (string, error) {
	file, err := os.CreateTemp(tmpdir, "")
	if err != nil {
		return "", err
	}

	defer multierr.AppendFunc(&err, file.Close)

	dl, _, err := fn(ctx, file, id)
	if err != nil {
		return "", err
	}

	logger.Info("Received document",
		zap.Int64("length_bytes", dl.Length),
		zap.String("suggested_filename", dl.Filename),
	)

	return file.Name(), nil
}

// ExtractVariantFacts downloads a particular document variant to a temporary
// directory before using an extraction function to get all facts. Among those,
// if any, the best are chosen and returned.
func ExtractVariantFacts(ctx context.Context, o ExtractVariantFactsOptions) (_ *paperminer.Facts, err error) {
	fn, err := selectDownloadFunction(o.Client, o.Variant)
	if err != nil {
		return nil, err
	}

	tmpdir, cleanup, err := fsutil.CreateTempdir(o.Basedir, fmt.Sprintf("doc-%d-%s-*", o.ID, o.Variant.String()))
	if err != nil {
		return nil, err
	}

	defer multierr.AppendFunc(&err, cleanup)

	path, err := download(ctx, o.Logger, tmpdir, fn, o.ID)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	all, err := o.Extract(ctx, o.Logger, path)
	if err != nil {
		return nil, fmt.Errorf("extracting facts from %q: %w", path, err)
	}

	return all.Best()
}
