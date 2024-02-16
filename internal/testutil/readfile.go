package testutil

import (
	"os"
	"testing"
)

func MustReadFileString(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("ReadFile(%q) failed: %v", path, err)
	}

	return string(content)
}
