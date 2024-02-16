package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/paperminer/internal/testutil"
)

func TestCreateTempdir(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dir     string
		pattern string
		wantErr error
	}{
		{
			name:    "success",
			pattern: "test-xyz-*",
		},
		{
			name:    "non-existent directory",
			dir:     filepath.Join(t.TempDir(), "nonexistent"),
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dir == "" {
				tc.dir = t.TempDir()
			}

			got, cleanup, err := CreateTempdir(tc.dir, tc.pattern)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if st := testutil.MustLstat(t, got); !st.IsDir() {
					t.Errorf("Path %s is not a directory: %+v", got, st)
				}

				if err := cleanup(); err != nil {
					t.Errorf("Cleanup failed: %v", err)
				}

				testutil.MustNotExist(t, got)
			}
		})
	}
}
