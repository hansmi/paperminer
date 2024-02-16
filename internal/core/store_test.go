package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestOpenDefaultStore(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dir     string
		wantErr error
	}{
		{
			name: "simple",
			dir:  t.TempDir(),
		},
		{
			name: "nonexistent directory",
			dir:  filepath.Join(t.TempDir(), "sub", "dir"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, cleanup, err := openDefaultStore(tc.dir)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if err := got.Bolt().Sync(); err != nil {
					t.Errorf("Sync() failed: %v", err)
				}

				if err := cleanup(); err != nil {
					t.Errorf("cleanup() failed: %v", err)
				}

				if entries, err := os.ReadDir(tc.dir); err != nil {
					t.Errorf("ReadDir() failed: %v", err)
				} else if len(entries) > 0 {
					t.Errorf("Directory %q is not empty: %q", tc.dir, entries)
				}
			}
		})
	}
}
