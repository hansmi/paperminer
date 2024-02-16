package document

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/ref"
	"go.uber.org/zap/zaptest"
)

func TestExtractFacts(t *testing.T) {
	errTest := errors.New("test error")

	for _, tc := range []struct {
		name    string
		opts    ExtractFactsOptions
		want    *paperminer.Facts
		wantErr error
	}{
		{
			name: "empty",
		},
		{
			name: "variants",
			opts: ExtractFactsOptions{
				Variants: []Variant{Archived, Original},
			},
		},
		{
			name: "first variant fails",
			opts: ExtractFactsOptions{
				Variants: []Variant{Original, Archived},
				Extract: func(_ context.Context, v Variant) (*paperminer.Facts, error) {
					if v == Original {
						return nil, errTest
					}

					return &paperminer.Facts{
						Title: ref.Ref("test title"),
					}, nil
				},
			},
			want: &paperminer.Facts{
				Title: ref.Ref("test title"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			tc.opts.Logger = zaptest.NewLogger(t)

			if tc.opts.Extract == nil {
				tc.opts.Extract = func(context.Context, Variant) (*paperminer.Facts, error) {
					return nil, nil
				}
			}

			got, err := ExtractFacts(ctx, tc.opts)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ExtractFacts() diff (-want +got):\n%s", diff)
			}
		})
	}
}
