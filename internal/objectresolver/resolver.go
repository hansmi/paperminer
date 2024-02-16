package objectresolver

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/singleflight"
)

type provider[T any] interface {
	kind() string
	create(ctx context.Context, name string) error
	listByName(ctx context.Context, name string) ([]T, error)
}

type Resolver[T any] struct {
	kind string
	zero T
	sf   singleflight.Group
	p    provider[T]
}

func newResolver[T any](p provider[T]) *Resolver[T] {
	return &Resolver[T]{
		kind: p.kind(),
		p:    p,
	}
}

func (r *Resolver[T]) once(key string, fn func() (T, error)) (T, error) {
	result, err, _ := r.sf.Do(key, func() (any, error) {
		return fn()
	})

	if err != nil {
		return r.zero, err
	}

	return result.(T), err
}

func (r *Resolver[T]) getFirst(name string, objs []T) (T, error) {
	if len(objs) == 0 {
		return r.zero, fmt.Errorf("%w: %s %q", ErrNotFound, r.kind, name)
	}

	if len(objs) > 1 {
		return r.zero, fmt.Errorf("%w: %s %q returned %d objects", ErrAmbiguous, r.kind, name, len(objs))
	}

	return objs[0], nil
}

func (r *Resolver[T]) getByName(ctx context.Context, name string) (T, error) {
	objs, err := r.p.listByName(ctx, name)
	if err != nil {
		return r.zero, err
	}

	return r.getFirst(name, objs)
}

func (r *Resolver[T]) GetByName(ctx context.Context, name string) (T, error) {
	return r.once(name, func() (T, error) {
		return r.getByName(ctx, name)
	})
}

func (r *Resolver[T]) GetOrCreateByName(ctx context.Context, name string) (T, error) {
	return r.once(name, func() (T, error) {
		obj, err := r.getByName(ctx, name)

		if errors.Is(err, ErrNotFound) {
			if err := r.p.create(ctx, name); err != nil {
				return r.zero, fmt.Errorf("creating %s %q: %w", r.kind, name, err)
			}

			obj, err = r.getByName(ctx, name)
		}

		if err != nil {
			return r.zero, fmt.Errorf("getting %s %q: %w", r.kind, name, err)
		}

		return obj, nil
	})
}
