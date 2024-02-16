package objectresolver

import (
	"testing"
)

func TestMemDocumentTypeResolver(t *testing.T) {
	validateMemResolver(t, NewMemDocumentTypeResolver())
}
