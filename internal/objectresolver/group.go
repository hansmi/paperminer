package objectresolver

import (
	"context"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type GroupClient interface {
	ListGroups(context.Context, plclient.ListGroupsOptions) ([]plclient.Group, *plclient.Response, error)
}

type groupProvider struct {
	GroupResolverOptions
}

func (groupProvider) kind() string {
	return "group"
}

func (p groupProvider) listByName(ctx context.Context, name string) ([]plclient.Group, error) {
	opts := plclient.ListGroupsOptions{}
	opts.Name.EqualsIgnoringCase = &name

	items, _, err := p.Client.ListGroups(ctx, opts)

	return items, err
}

func (p groupProvider) create(ctx context.Context, name string) error {
	return ErrCreateUnsupported
}

type GroupResolver = Resolver[plclient.Group]

type GroupResolverOptions struct {
	Client GroupClient
}

func NewGroupResolver(opts GroupResolverOptions) *GroupResolver {
	return newResolver[plclient.Group](&groupProvider{opts})
}

func NewMemGroupResolver() *GroupResolver {
	return newMemResolver(func(id int64, name string) plclient.Group {
		return plclient.Group{
			ID:   id,
			Name: name,
		}
	})
}
