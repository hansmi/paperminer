package objectresolver

import (
	"context"
	"testing"
)

func MustGetOrCreateByName[T any](t *testing.T, resolver *Resolver[T], name string) T {
	t.Helper()

	item, err := resolver.GetOrCreateByName(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}

	return item
}
