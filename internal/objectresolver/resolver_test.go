package objectresolver

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestResolver(t *testing.T) {
	ctx := context.Background()

	p := newMemProvider(func(id int64, name string) string {
		return fmt.Sprintf("value %s", name)
	})

	r := newResolver[string](p)

	if _, err := r.GetByName(ctx, "1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Missing item not reported as such: %v", err)
	}

	p.set("1", "value")

	if got, err := r.GetByName(ctx, "1"); err != nil {
		t.Errorf("GetByName() failed: %v", err)
	} else if want := "value"; got != want {
		t.Errorf("GetByName() returned %q, want %q", got, want)
	}

	if got, err := r.GetOrCreateByName(ctx, "2"); err != nil {
		t.Errorf("GetOrCreateByName() failed: %v", err)
	} else if want := "value 2"; got != want {
		t.Errorf("GetOrCreateByName() returned %q, want %q", got, want)
	}
}
