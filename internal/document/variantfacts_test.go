package document

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/facter"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type fakeVariantFactsClient struct {
	originalErr error
	archivedErr error
}

func (c *fakeVariantFactsClient) DownloadDocumentOriginal(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error) {
	return &plclient.DownloadResult{}, nil, c.originalErr
}

func (c *fakeVariantFactsClient) DownloadDocumentArchived(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error) {
	return &plclient.DownloadResult{}, nil, c.archivedErr
}

func TestExtractVariantFacts(t *testing.T) {
	errTest := errors.New("test error")

	for _, tc := range []struct {
		name    string
		cl      VariantFactsClient
		extract ExtractFileFactsFunc
		variant Variant
		want    *paperminer.Facts
		wantErr error
	}{
		{name: "defaults"},
		{
			name: "download fails",
			cl: &fakeVariantFactsClient{
				archivedErr: errors.New("another"),
				originalErr: errTest,
			},
			variant: Original,
			wantErr: errTest,
		},
		{
			name: "facts",
			extract: func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
				return facter.FactsSlice{{
					Title: plclient.String("Test"),
				}}, nil
			},
			variant: Archived,
			want: &paperminer.Facts{
				Title: plclient.String("Test"),
			},
		},
		{
			name: "no facts",
			extract: func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
				return nil, nil
			},
			variant: Archived,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			if tc.cl == nil {
				tc.cl = &fakeVariantFactsClient{}
			}

			if tc.extract == nil {
				tc.extract = func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
					return nil, nil
				}
			}

			opts := ExtractVariantFactsOptions{
				Logger:  zaptest.NewLogger(t),
				Basedir: t.TempDir(),
				Client:  tc.cl,
				Extract: tc.extract,
				Variant: tc.variant,
			}

			got, err := ExtractVariantFacts(ctx, opts)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Facts diff (-want +got):\n%s", diff)
			}
		})
	}
}
