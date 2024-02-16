package objectresolver

import (
	"testing"
)

func TestMemCorrespondentResolver(t *testing.T) {
	validateMemResolver(t, NewMemCorrespondentResolver())
}
