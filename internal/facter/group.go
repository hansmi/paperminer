package facter

import (
	"context"
	"fmt"
	"runtime"

	"github.com/hansmi/dossier"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/ref"
	"github.com/hansmi/staticplug"
	"github.com/sourcegraph/conc/stream"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var documentFacterType = staticplug.MustTypeOfInterface((*paperminer.DocumentFacter)(nil))

func GroupFromRegistry(reg *staticplug.Registry) (*Group, error) {
	plugins, err := reg.PluginsImplementing(documentFacterType)
	if err != nil {
		return nil, err
	}

	g := &Group{}

	for _, p := range plugins {
		if inst, err := p.New(); err != nil {
			return nil, fmt.Errorf("instantiating plugin %q: %w", p.Name, err)
		} else {
			g.plugins = append(g.plugins, newPluginWrapper(inst.(paperminer.DocumentFacter)))
		}
	}

	return g, nil
}

type Group struct {
	plugins []*pluginWrapper
}

func (g *Group) IsEmpty() bool {
	return len(g.plugins) == 0
}

func (g *Group) Names() []string {
	result := make([]string, 0, len(g.plugins))

	for _, w := range g.plugins {
		result = append(result, w.name)
	}

	return result
}

func (g *Group) Extract(ctx context.Context, logger *zap.Logger, doc *dossier.Document) (FactsSlice, error) {
	var result FactsSlice
	var resultErr error

	s := stream.New().WithMaxGoroutines(runtime.GOMAXPROCS(0))

	for _, w := range g.plugins {
		w := w
		s.Go(func() stream.Callback {
			facts, err := w.inst.DocumentFacts(ctx, paperminer.DocumentFacterOptions{
				Logger:   logger.With(zap.String("plugin", w.name)),
				Document: doc,
			})

			return func() {
				if err != nil {
					multierr.AppendInto(&resultErr, fmt.Errorf("plugin %q: %w", w.name, err))
				} else if !(facts == nil || facts.IsEmpty()) {
					if facts.Reporter == nil {
						facts.Reporter = ref.Ref(w.name)
					}

					result = append(result, facts)
				}
			}
		})
	}

	s.Wait()

	logger.Debug("Fact extraction complete",
		zap.Int("count", len(result)),
		zap.Any("facts", result),
	)

	return result, resultErr
}
