package objectresolver

import (
	"testing"
)

func TestMemStoragePathResolver(t *testing.T) {
	validateMemResolver(t, NewMemStoragePathResolver())
}
