package document

import (
	"context"
	"fmt"

	"github.com/hansmi/paperminer"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type ExtractVariantFactsFunc func(context.Context, Variant) (*paperminer.Facts, error)

type ExtractFactsOptions struct {
	Logger   *zap.Logger
	Variants []Variant
	Extract  ExtractVariantFactsFunc
}

func ExtractFacts(ctx context.Context, o ExtractFactsOptions) (*paperminer.Facts, error) {
	var allErr error

	for _, v := range o.Variants {
		facts, err := o.Extract(ctx, v)
		if err != nil {
			multierr.AppendInto(&allErr, fmt.Errorf("variant %q: %w", v.String(), err))
			continue
		}

		if !(facts == nil || facts.IsEmpty()) {
			if allErr != nil {
				o.Logger.Debug("Fact extraction successful after earlier failure",
					zap.Stringer("success_variant", v),
					zap.NamedError("previous_errors", allErr),
				)
			}

			return facts, nil
		}
	}

	return nil, allErr
}
