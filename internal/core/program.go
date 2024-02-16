package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperhooks/pkg/kpflag"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/cataloger"
	"github.com/hansmi/paperminer/internal/httpsrv"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"github.com/hansmi/paperminer/internal/workflow"
	"github.com/hansmi/staticplug"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var workflowCreateFuncs = map[string]workflow.CreateFunc{
	"cataloger":   cataloger.New,
	"storepruner": newStorePruner,
}

func newMux() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	return r
}

type Program struct {
	name string

	logger *zap.Logger

	pluginRegistry          *staticplug.Registry
	metricsRegistry         *prometheus.Registry
	prefixedMetricsRegistry prometheus.Registerer
	mux                     *chi.Mux

	storeDir          string
	listenAddress     string
	clientFlags       plclient.Flags
	objectPermissions objectresolver.NamedObjectPermissions

	workflowEnvBase *workflowEnvBase
	workflows       []workflow.Workflow
}

func NewProgram(ctx context.Context, logger *zap.Logger, app *kingpin.Application) (*Program, error) {
	p := &Program{
		name:   filepath.Base(os.Args[0]),
		logger: logger,

		pluginRegistry:  paperminer.GlobalPluginRegistry(),
		metricsRegistry: prometheus.NewPedanticRegistry(),
	}
	p.registerFlags(app)
	p.setupMux()

	p.prefixedMetricsRegistry = prometheus.WrapRegistererWithPrefix(fmt.Sprintf("%s_", p.name), p.metricsRegistry)

	p.workflowEnvBase = &workflowEnvBase{
		p:   p,
		app: app,
	}

	if workflows, err := p.setupWorkflows(ctx); err != nil {
		return nil, err
	} else {
		p.workflows = workflows
	}

	return p, nil
}

func (p *Program) registerFlags(app *kingpin.Application) {
	app.Flag("listen_address", "Address and port on which to expose the HTTP API.").
		Default(net.JoinHostPort(netip.IPv6Loopback().String(), "0")).
		StringVar(&p.listenAddress)

	kpflag.RegisterClient(app, &p.clientFlags)

	p.objectPermissions.RegisterFlags(app)
}

func (p *Program) setupMux() {
	p.mux = newMux()
	p.mux.Method(http.MethodGet, "/metrics",
		promhttp.HandlerFor(p.metricsRegistry, promhttp.HandlerOpts{
			Registry: p.metricsRegistry,
		}))

	p.mux.Post("/notify/post-consumption", func(w http.ResponseWriter, r *http.Request) {
		p.notifyPostConsumption()
		w.WriteHeader(http.StatusNoContent)
	})
}

func (p *Program) setupWorkflows(ctx context.Context) ([]workflow.Workflow, error) {
	var result []workflow.Workflow

	for name, create := range workflowCreateFuncs {
		wf, err := create(ctx, &workflowEnv{
			workflowEnvBase: p.workflowEnvBase,
			logger:          p.logger.With(zap.String("workflow", name)),
		})
		if err != nil {
			return nil, fmt.Errorf("creating workflow %q: %w", name, err)
		}

		result = append(result, wf)
	}

	return result, nil
}

func (p *Program) notifyPostConsumption() {
	for _, wf := range p.workflows {
		if nr := wf.(workflow.NotificationReceiver); nr != nil {
			nr.NotifyPostConsumption()
		}
	}
}

func (p *Program) Run(ctx context.Context) (err error) {
	p.metricsRegistry.MustRegister(
		collectors.NewBuildInfoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
		version.NewCollector(p.name),
	)

	for _, wf := range p.workflows {
		if err := wf.Validate(ctx); errors.Is(err, workflow.ErrValidationEarlyExit) {
			return nil
		} else if err != nil {
			return err
		}
	}

	client, err := p.clientFlags.Build()
	if err != nil {
		return err
	}

	s, storeCleanup, err := openDefaultStore(p.storeDir)
	if err != nil {
		return err
	}

	defer multierr.AppendFunc(&err, storeCleanup)

	resolvers, err := objectresolver.NewObjectResolvers(ctx, client, p.objectPermissions)
	if err != nil {
		return err
	}

	envBase := p.workflowEnvBase
	envBase.mu.Lock()
	envBase.client = client
	envBase.store = s
	envBase.resolvers = resolvers
	envBase.mu.Unlock()

	httpReadyCh := make(chan net.Addr)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpsrv.ListenAndServe(ctx, httpsrv.ListenAndServeOptions{
			Logger:  p.logger,
			Address: p.listenAddress,
			Handler: p.mux,
			ReadyCh: httpReadyCh,
		})
	})

	g.Go(func() error {
		// Wait for HTTP server to be ready before starting workflows.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-httpReadyCh:
		}

		for _, wf := range p.workflows {
			wf := wf

			g.Go(func() error {
				return wf.Run(ctx)
			})
		}

		return nil
	})

	return g.Wait()
}
