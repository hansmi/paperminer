package facter

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/ref"
)

func TestFactsSliceBest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		s       FactsSlice
		want    *paperminer.Facts
		wantErr error
	}{
		{
			name: "empty",
		},
		{
			name: "one",
			s: []*paperminer.Facts{{
				Reporter: ref.Ref("xyz"),
				Title:    ref.Ref("title"),
			}},
			want: &paperminer.Facts{
				Reporter: ref.Ref("xyz"),
				Title:    ref.Ref("title"),
			},
		},
		{
			name:    "multiple",
			s:       []*paperminer.Facts{{}, {}, {}},
			wantErr: cmpopts.AnyError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.s.Best()

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Facts diff (-want +got):\n%s", diff)
			}
		})
	}
}
