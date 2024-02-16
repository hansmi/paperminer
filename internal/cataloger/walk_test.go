package cataloger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type fakeWalkDocumentsClient struct {
	docs []plclient.Document
}

func (c *fakeWalkDocumentsClient) ListAllDocuments(ctx context.Context, opts plclient.ListDocumentsOptions, handler func(context.Context, plclient.Document) error) error {
	for _, i := range c.docs[:] {
		if err := handler(ctx, i); err != nil {
			return err
		}

		c.docs = append(c.docs, plclient.Document{ID: 900})
	}

	return nil
}

func TestWalkDocuments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	client := &fakeWalkDocumentsClient{
		docs: []plclient.Document{
			{ID: 100},
			{ID: 200},
			{ID: 300},
		},
	}

	var mu sync.Mutex
	var handled []int64

	handler := func(_ context.Context, _ *zap.Logger, doc *plclient.Document) error {
		mu.Lock()
		handled = append(handled, doc.ID)
		mu.Unlock()
		return nil
	}

	if err := walkDocuments(ctx, zaptest.NewLogger(t), client, 0, handler); err != nil {
		t.Errorf("walkDocuments() failed: %v", err)
	}

	want := []int64{100, 200, 300, 900}

	if diff := cmp.Diff(want, handled, cmpopts.EquateEmpty(), testutil.CmpSortInt64Slices); diff != "" {
		t.Errorf("Handled documents diff (-want +got):\n%s", diff)
	}
}
