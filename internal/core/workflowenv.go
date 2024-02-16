package core

import (
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-chi/chi/v5"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"github.com/hansmi/paperminer/internal/workflow"
	"github.com/hansmi/staticplug"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
)

type workflowEnvBase struct {
	p *Program

	app *kingpin.Application

	mu        sync.Mutex
	store     *bolthold.Store
	client    *plclient.Client
	resolvers *objectresolver.ObjectResolvers
}

func (e *workflowEnv) ProgramName() string {
	return e.p.name
}

func (e *workflowEnvBase) App() *kingpin.Application {
	return e.app
}

func (e *workflowEnvBase) PluginRegistry() *staticplug.Registry {
	return e.p.pluginRegistry
}

func (e *workflowEnvBase) MetricsRegistry() prometheus.Registerer {
	return e.p.metricsRegistry
}

func (e *workflowEnvBase) Mux() *chi.Mux {
	return e.p.mux
}

func (e *workflowEnvBase) Store() *bolthold.Store {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.store == nil {
		panic("Store not yet available")
	}

	return e.store
}

func (e *workflowEnvBase) Client() *plclient.Client {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.client == nil {
		panic("Client not yet available")
	}

	return e.client
}

func (e *workflowEnvBase) Resolvers() *objectresolver.ObjectResolvers {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.resolvers == nil {
		panic("Resolvers not yet available")
	}

	return e.resolvers
}

type workflowEnv struct {
	*workflowEnvBase
	logger *zap.Logger
}

var _ workflow.Environment = (*workflowEnv)(nil)

func (e *workflowEnv) Logger() *zap.Logger {
	return e.logger
}
