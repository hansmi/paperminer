package sketchfacts

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hansmi/dossier/pkg/sketch"
	"github.com/hansmi/dossier/pkg/sketchiter"
	"github.com/hansmi/dossier/proto/sketchpb"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/staticplug"
	"go.uber.org/zap"
)

var errInvalidReport = errors.New("invalid report")

type BuildFunc func(*sketch.PageReport) (*paperminer.Facts, error)

type Options struct {
	Name string

	// Sketch definition in textproto format.
	Textproto string

	// Sketch definition as a protocol buffer message.
	Sketch *sketchpb.Sketch

	// Nodes which must be valid on the first page.
	Required []string

	// Function to build facts from report. Return nil facts to indicate that
	// a document wasn't recognized.
	Build BuildFunc
}

type Plugin struct {
	opts Options
	s    *sketch.Sketch
}

var _ staticplug.Plugin = (*Plugin)(nil)
var _ paperminer.DocumentFacter = (*Plugin)(nil)

// New creates a facter using a dossier sketch to evaluate the first page of
// a document. Pages beyond the first are ignored.
func New(opts Options) (*Plugin, error) {
	var err error
	var s *sketch.Sketch

	if opts.Textproto != "" && opts.Sketch == nil {
		s, err = sketch.CompileFromTextprotoString(opts.Textproto)
	} else if opts.Textproto == "" && opts.Sketch != nil {
		s, err = sketch.Compile(opts.Sketch)
	} else {
		return nil, fmt.Errorf(`%w: exactly one of "Textproto" and "Sketch" may be set`, os.ErrInvalid)
	}

	if err != nil {
		return nil, err
	}

	if opts.Build == nil {
		return nil, fmt.Errorf("%w: build function is required", os.ErrInvalid)
	}

	// TODO: Check sketch for the required nodes. For that to be possible the
	// sketch needs to expose the information without analyzing.

	return &Plugin{
		opts: opts,
		s:    s,
	}, nil
}

func MustNew(opts Options) *Plugin {
	p, err := New(opts)
	if err != nil {
		panic(err)
	}
	return p
}

func (p *Plugin) PluginInfo() staticplug.PluginInfo {
	return staticplug.PluginInfo{
		Name: p.opts.Name,
		New: func() (staticplug.Plugin, error) {
			// Instances are stateless.
			return p, nil
		},
	}
}

func (p *Plugin) validate(logger *zap.Logger, report *sketch.PageReport) (bool, error) {
	for _, name := range p.opts.Required {
		if node := report.NodeByName(name); node == nil {
			return false, fmt.Errorf("%w: node %q not found", errInvalidReport, name)
		} else if !node.Valid() {
			logger.Debug("Node not valid on first page", zap.String("node", name))
			return false, nil
		}
	}

	return true, nil
}

func (p *Plugin) DocumentFacts(ctx context.Context, opts paperminer.DocumentFacterOptions) (*paperminer.Facts, error) {
	it := sketchiter.NewPageIter(p.s, opts.Document)

	report, err := it.Next(ctx)

	if errors.Is(err, sketchiter.Done) {
		// Empty document
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if ok, err := p.validate(opts.Logger, report); err != nil || !ok {
		return nil, err
	}

	facts, err := p.opts.Build(report)
	if err != nil {
		return nil, fmt.Errorf("building facts: %w", err)
	}

	return facts, nil
}
