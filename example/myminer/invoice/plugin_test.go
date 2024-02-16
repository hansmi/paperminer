package invoice

import (
	"testing"

	"github.com/hansmi/paperminer/pkg/factertest"
)

func Test(t *testing.T) {
	tc := factertest.NewTestCase(t, Plugin)
	tc.Assert(t, "testdata/invoice.pdf")
}
