package document

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/dossier"
	"github.com/hansmi/dossier/pkg/parsertest"
	"github.com/hansmi/paperminer/internal/facter"
	"github.com/hansmi/paperminer/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestMakeFileFactsExtractor(t *testing.T) {
	emptyFile := testutil.MustWriteFileString(t, filepath.Join(t.TempDir(), "empty"), "")

	for _, tc := range []struct {
		name    string
		path    string
		opts    []dossier.DocumentOption
		extract ExtractDocFactsFunc
		wantErr error
		want    facter.FactsSlice
	}{
		{
			name:    "missing file",
			path:    filepath.Join(t.TempDir(), "missing"),
			wantErr: os.ErrNotExist,
		},
		{
			name: "empty document",
			path: emptyFile,
			opts: []dossier.DocumentOption{
				dossier.WithStaticDocumentParser(&parsertest.SimpleParser{}),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			if tc.extract == nil {
				tc.extract = func(context.Context, *zap.Logger, *dossier.Document) (facter.FactsSlice, error) {
					return nil, nil
				}
			}

			got, err := MakeFileFactsExtractor(tc.extract, tc.opts...)(ctx, zaptest.NewLogger(t), tc.path)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Facts diff (-want +got):\n%s", diff)
			}
		})
	}
}
