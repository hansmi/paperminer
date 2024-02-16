package fsutil

import (
	"os"
)

type CleanupFunc func() error

// CreateTempdir wraps [os.MkdirTemp] and returns an additional cleanup
// function to automatically remove the created directory.
func CreateTempdir(dir, pattern string) (string, CleanupFunc, error) {
	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", nil, err
	}

	return path, func() error {
		return os.RemoveAll(path)
	}, nil
}
