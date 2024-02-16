package objectresolver

import (
	"context"
	"errors"
	"testing"
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
