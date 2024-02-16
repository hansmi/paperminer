package objectresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func validateMemResolver[T any](t *testing.T, r *Resolver[T]) {
	t.Helper()

	ctx := context.Background()

	for _, name := range []string{"", "foo", "bar"} {
		if got, err := r.GetByName(ctx, name); !errors.Is(err, ErrNotFound) {
			t.Errorf("GetByName(%q) returned unexpected error (%#v): %v", name, got, err)
		}

		if _, err := r.GetOrCreateByName(ctx, name); err != nil {
			t.Errorf("GetOrCreateByName(%q) should have created an object, failed instead: %v", name, err)
		}

		if _, err := r.GetByName(ctx, name); err != nil {
			t.Errorf("GetByName(%q) failed: %v", name, err)
		}
	}

}

func TestMemProviderKind(t *testing.T) {
	p := newMemProvider(func(id int64, name string) bool {
		return false
	})

	want := "memProvider[bool]"

	if diff := cmp.Diff(want, p.kind()); diff != "" {
		t.Errorf("kind() diff (-want +got):\n%s", diff)
	}
}
