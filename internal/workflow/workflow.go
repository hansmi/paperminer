package workflow

import (
	"context"
	"errors"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-chi/chi/v5"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"github.com/hansmi/staticplug"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
)

var ErrValidationEarlyExit = errors.New("successful exit from validation")

type Environment interface {
	ProgramName() string
	Logger() *zap.Logger
	App() *kingpin.Application

	PluginRegistry() *staticplug.Registry
	MetricsRegistry() prometheus.Registerer

	Mux() *chi.Mux

	Store() *bolthold.Store
	Client() *plclient.Client
	Resolvers() *objectresolver.ObjectResolvers
}

type CreateFunc func(context.Context, Environment) (Workflow, error)

type Workflow interface {
	Validate(context.Context) error
	Run(context.Context) error
}

type NotificationReceiver interface {
	NotifyPostConsume()
}
