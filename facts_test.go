package paperminer

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hansmi/paperminer/internal/ref"
)

func TestFactsIsEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *Facts
		want  bool
	}{
		{
			name:  "zero",
			value: &Facts{},
			want:  true,
		},
		{
			name:  "title",
			value: &Facts{Title: ref.Ref("xyz")},
		},
		{
			name:  "created",
			value: &Facts{Created: ref.Ref(time.Now())},
		},
		{
			name:  "set tags",
			value: &Facts{SetTags: []string{"x"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, tc.value.IsEmpty()); diff != "" {
				t.Errorf("IsEmpty() diff (-want +got):\n%s", diff)
			}
		})
	}
}
