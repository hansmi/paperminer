package cataloger

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/document"
	"github.com/hansmi/paperminer/internal/facter"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"github.com/hansmi/paperminer/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestUpdaterCheckSize(t *testing.T) {
	const fileSizeMax = 1024 * 1024

	for _, tc := range []struct {
		name     string
		metadata plclient.DocumentMetadata
		wantErr  error
	}{
		{
			name: "empty",
		},
		{
			name: "original too large",
			metadata: plclient.DocumentMetadata{
				OriginalSize: 3 * fileSizeMax,
			},
			wantErr: errDocumentTooLarge,
		},
		{
			name: "archive too large",
			metadata: plclient.DocumentMetadata{
				HasArchiveVersion: true,
				ArchiveSize:       100 * fileSizeMax,
			},
			wantErr: errDocumentTooLarge,
		},
		{
			name: "missing archive too large",
			metadata: plclient.DocumentMetadata{
				HasArchiveVersion: false,
				ArchiveSize:       100 * fileSizeMax,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			u, err := newUpdater(ctx, updaterOptions{
				Logger:      zaptest.NewLogger(t),
				Resolvers:   objectresolver.NewMemObjectResolvers(),
				Metadata:    &tc.metadata,
				FileSizeMax: fileSizeMax,
			})

			if err != nil {
				t.Errorf("newUpdater() failed: %v", err)
			}

			err = u.checkSize()

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}

type fakeUpdaterClient struct {
	patches []map[string]any
}

func (c *fakeUpdaterClient) DownloadDocumentOriginal(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error) {
	return &plclient.DownloadResult{}, nil, nil
}

func (c *fakeUpdaterClient) DownloadDocumentArchived(context.Context, io.Writer, int64) (*plclient.DownloadResult, *plclient.Response, error) {
	return &plclient.DownloadResult{}, nil, nil
}

func (c *fakeUpdaterClient) PatchDocument(_ context.Context, _ int64, fields *plclient.DocumentFields) (*plclient.Document, *plclient.Response, error) {
	c.patches = append(c.patches, fields.AsMap())

	return nil, nil, nil
}

func TestUpdater(t *testing.T) {
	const fileSizeMax = 1024 * 1024

	errTest := errors.New("test")

	resolvers := objectresolver.NewMemObjectResolvers()

	todoTag := objectresolver.MustGetOrCreateByName(t, resolvers.Tag, "xyz todo")
	failedTag := objectresolver.MustGetOrCreateByName(t, resolvers.Tag, "pfx:failed")
	customTag := objectresolver.MustGetOrCreateByName(t, resolvers.Tag, "custom abc")

	for _, tc := range []struct {
		name        string
		doc         plclient.Document
		metadata    plclient.DocumentMetadata
		extract     document.ExtractFileFactsFunc
		lastRetry   bool
		wantErr     error
		wantPatches []map[string]any
	}{
		{
			name: "defaults",
		},
		{
			name: "extraction fails",
			extract: func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
				return nil, errTest
			},
			wantErr: errTest,
		},
		{
			name: "extraction fails on last retry",
			extract: func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
				return nil, errTest
			},
			lastRetry: true,
			wantPatches: []map[string]any{{
				"tags": []int64{failedTag.ID},
			}},
		},
		{
			name: "facts",
			doc: plclient.Document{
				Title:        "original title",
				DocumentType: plclient.Int64(1),
			},
			extract: func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
				return facter.FactsSlice{{
					Title:        plclient.String(""),
					DocumentType: plclient.String(""),
					SetTags:      []string{"custom abc"},
				}}, nil
			},
			wantPatches: []map[string]any{{
				"title":         "",
				"document_type": (*int64)(nil),
				"tags":          []int64{customTag.ID},
			}},
		},
		{
			name: "file size too large",
			metadata: plclient.DocumentMetadata{
				OriginalSize:      fileSizeMax,
				HasArchiveVersion: true,
				ArchiveSize:       fileSizeMax + 1,
			},
			wantPatches: []map[string]any{{
				"tags": []int64{failedTag.ID},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			client := &fakeUpdaterClient{}

			if tc.extract == nil {
				tc.extract = func(context.Context, *zap.Logger, string) (facter.FactsSlice, error) {
					return nil, nil
				}
			}

			u, err := newUpdater(ctx, updaterOptions{
				Logger:           zaptest.NewLogger(t),
				Resolvers:        resolvers,
				Client:           client,
				Document:         &tc.doc,
				Metadata:         &tc.metadata,
				TodoTagName:      todoTag.Name,
				FailedTagName:    failedTag.Name,
				FileSizeMax:      fileSizeMax,
				ExtractFileFacts: tc.extract,
				CheckModified: func(context.Context) error {
					return nil
				},
			})

			if err != nil {
				t.Errorf("newUpdater() failed: %v", err)
			}

			err = u.Do(ctx, tc.lastRetry)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantPatches, client.patches, cmpopts.EquateEmpty(), testutil.CmpSortInt64Slices); diff != "" {
				t.Errorf("Patches diff (-want +got):\n%s", diff)
			}
		})
	}
}
