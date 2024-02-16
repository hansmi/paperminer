package document

import (
	"context"
	"fmt"

	"github.com/hansmi/dossier"
	"github.com/hansmi/paperminer/internal/facter"
	"go.uber.org/zap"
)

type ExtractDocFactsFunc func(context.Context, *zap.Logger, *dossier.Document) (facter.FactsSlice, error)

func MakeFileFactsExtractor(extract ExtractDocFactsFunc, opts ...dossier.DocumentOption) ExtractFileFactsFunc {
	return func(ctx context.Context, logger *zap.Logger, path string) (facter.FactsSlice, error) {
		doc := dossier.NewDocument(path, opts...)

		if err := doc.Validate(ctx); err != nil {
			return nil, fmt.Errorf("file validation: %w", err)
		}

		all, err := extract(ctx, logger, doc)
		if err != nil {
			return nil, fmt.Errorf("fact extraction: %w", err)
		}

		return all, nil
	}
}
