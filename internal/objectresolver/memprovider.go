package objectresolver

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
)

const memProviderMinID = 1000

// memProvider is an in-memory provider.
type memProvider[T any] struct {
	mu      sync.Mutex
	objects map[string]T

	createObject func(string) T
}

var _ provider[struct{}] = (*memProvider[struct{}])(nil)

func (p *memProvider[T]) kind() string {
	return reflect.TypeOf(p).Elem().Name()
}

func (p *memProvider[T]) set(name string, value T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.objects[name] = value
}

func (p *memProvider[T]) create(_ context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.objects[name]; ok {
		return fmt.Errorf("key %q exists already", name)
	}

	p.objects[name] = p.createObject(name)

	return nil
}

func (p *memProvider[T]) listByName(_ context.Context, name string) ([]T, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if v, ok := p.objects[name]; ok {
		return []T{v}, nil
	}

	return nil, nil
}

func newMemProvider[T any](create func(id int64, name string) T) *memProvider[T] {
	var id atomic.Int64

	id.Store(memProviderMinID + rand.Int63n(1<<24))

	return &memProvider[T]{
		objects: map[string]T{},
		createObject: func(name string) T {
			return create(id.Add(1), name)
		},
	}
}

// create is an optional function to create a new value for a key. By
// default values are set to their zero state.
func newMemResolver[T any](create func(id int64, name string) T) *Resolver[T] {
	return newResolver[T](newMemProvider[T](create))
}
