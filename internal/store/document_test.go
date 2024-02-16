package store

import (
	"regexp"
	"testing"
	"time"
)

func TestDocumentTaskKey(t *testing.T) {
	for _, tc := range []struct {
		name       string
		task       DocumentTask
		want       *regexp.Regexp
		wantMinLen int
	}{
		{
			name:       "empty",
			want:       regexp.MustCompile(`\0""""$`),
			wantMinLen: 25,
		},
		{
			name: "populated",
			task: DocumentTask{
				ID:               20429,
				Added:            time.Date(2023, time.January, 1, 1, 0, 0, 0, time.UTC),
				Modified:         time.Date(2024, time.January, 1, 1, 0, 0, 0, time.UTC),
				OriginalChecksum: "52233939",
				ArchiveChecksum:  "2482243518533",
			},
			want:       regexp.MustCompile(`\0"52233939""2482243518533"$`),
			wantMinLen: 45,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.task.Key()

			if err != nil {
				t.Errorf("Key() failed: %v", err)
			} else if len(got) < tc.wantMinLen {
				t.Errorf("Got %d bytes, want at least %d: %q", len(got), tc.wantMinLen, got)
			} else if !tc.want.Match(got) {
				t.Errorf("Key %q doesn't match pattern %q", got, tc.want.String())
			}
		})
	}
}
