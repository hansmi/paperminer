package cataloger

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"github.com/hansmi/paperminer/internal/ref"
	"github.com/hansmi/paperminer/internal/testutil"
)

func TestPatchBuilder(t *testing.T) {
	resolvers := objectresolver.NewMemObjectResolvers()

	firstTag := objectresolver.MustGetOrCreateByName(t, resolvers.Tag, "first tag")
	secondTag := objectresolver.MustGetOrCreateByName(t, resolvers.Tag, "second tag")

	firstCorrespondent := objectresolver.MustGetOrCreateByName(t, resolvers.Correspondent, "first correspondent")
	firstDocumentType := objectresolver.MustGetOrCreateByName(t, resolvers.DocumentType, "first documenttype")
	firstStoragePath := objectresolver.MustGetOrCreateByName(t, resolvers.StoragePath, "first storagepath")

	for _, tc := range []struct {
		name         string
		doc          plclient.Document
		facts        *paperminer.Facts
		want         map[string]any
		wantFactsErr error
	}{
		{
			name: "empty",
			want: map[string]any{},
		},
		{
			name: "unmodified",
			doc: plclient.Document{
				Title:         "test title",
				Created:       time.Now(),
				Correspondent: plclient.Int64(1),
				DocumentType:  plclient.Int64(2),
				StoragePath:   plclient.Int64(3),
				Tags:          []int64{7, 8, 9, 1, 2, 3},
			},
			want: map[string]any{},
		},
		{
			name: "created and title",
			doc: plclient.Document{
				Title: "original",
			},
			facts: &paperminer.Facts{
				Title:   ref.Ref("changed"),
				Created: ref.Ref(time.Date(2020, time.January, 1, 1, 2, 3, 0, time.UTC)),
			},
			want: map[string]any{
				"title":   "changed",
				"created": time.Date(2020, time.January, 1, 1, 2, 3, 0, time.UTC),
			},
		},
		{
			name: "created and title unchanged",
			doc: plclient.Document{
				Title:   "hello",
				Created: time.Date(2020, time.March, 4, 5, 6, 7, 0, time.UTC),
			},
			facts: &paperminer.Facts{
				Title:   ref.Ref("hello"),
				Created: ref.Ref(time.Date(2020, time.March, 4, 5, 6, 7, 0, time.UTC)),
			},
			want: map[string]any{},
		},
		{
			name: "objects",
			facts: &paperminer.Facts{
				Correspondent: ref.Ref(firstCorrespondent.Name),
				DocumentType:  ref.Ref(firstDocumentType.Name),
				StoragePath:   ref.Ref(firstStoragePath.Name),
			},
			want: map[string]any{
				"correspondent": &firstCorrespondent.ID,
				"document_type": &firstDocumentType.ID,
				"storage_path":  &firstStoragePath.ID,
			},
		},
		{
			name: "unset objects",
			doc: plclient.Document{
				Correspondent: &firstCorrespondent.ID,
				DocumentType:  &firstDocumentType.ID,
				StoragePath:   &firstStoragePath.ID,
			},
			facts: &paperminer.Facts{
				Correspondent: ref.Ref(""),
				DocumentType:  ref.Ref(""),
				StoragePath:   ref.Ref(""),
			},
			want: map[string]any{
				"correspondent": (*int64)(nil),
				"document_type": (*int64)(nil),
				"storage_path":  (*int64)(nil),
			},
		},
		{
			name: "facts with tags",
			doc: plclient.Document{
				Tags: []int64{0, math.MaxInt64, secondTag.ID},
			},
			facts: &paperminer.Facts{
				SetTags:   []string{firstTag.Name, firstTag.Name},
				UnsetTags: []string{secondTag.Name},
			},
			want: map[string]any{
				"tags": []int64{0, firstTag.ID, math.MaxInt64},
			},
		},
		{
			name: "tags set to empty",
			doc: plclient.Document{
				Tags: []int64{firstTag.ID, secondTag.ID},
			},
			facts: &paperminer.Facts{
				UnsetTags: []string{firstTag.Name, secondTag.Name},
			},
			want: map[string]any{
				"tags": []int64{},
			},
		},
		{
			name: "unset unknown tag",
			facts: &paperminer.Facts{
				UnsetTags: []string{"unknown tag 1234"},
			},
			wantFactsErr: objectresolver.ErrNotFound,
			want:         map[string]any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			pb := newPatchBuilder(resolvers, &tc.doc)

			if tc.facts != nil {
				err := pb.setFacts(ctx, tc.facts)

				if diff := cmp.Diff(tc.wantFactsErr, err, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("setFacts() error diff (-want +got):\n%s", diff)
				}
			}

			if diff := cmp.Diff(tc.want, pb.build().AsMap(), testutil.CmpSortInt64Slices); diff != "" {
				t.Errorf("Patch diff (-want +got):\n%s", diff)
			}
		})
	}
}
