package objectresolver

import (
	"context"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type CorrespondentClient interface {
	ListCorrespondents(context.Context, plclient.ListCorrespondentsOptions) ([]plclient.Correspondent, *plclient.Response, error)
	CreateCorrespondent(context.Context, *plclient.CorrespondentFields) (*plclient.Correspondent, *plclient.Response, error)
}

type correspondentProvider struct {
	CorrespondentResolverOptions
}

func (correspondentProvider) kind() string {
	return "correspondent"
}

func (p *correspondentProvider) create(ctx context.Context, name string) error {
	fields := plclient.NewCorrespondentFields().SetName(name)

	p.PermissionOptions.apply(fields)

	_, _, err := p.Client.CreateCorrespondent(ctx, fields)

	return err
}

func (p *correspondentProvider) listByName(ctx context.Context, name string) ([]plclient.Correspondent, error) {
	opts := plclient.ListCorrespondentsOptions{}
	opts.Name.EqualsIgnoringCase = &name

	items, _, err := p.Client.ListCorrespondents(ctx, opts)

	return items, err
}

type CorrespondentResolver = Resolver[plclient.Correspondent]

type CorrespondentResolverOptions struct {
	PermissionOptions

	Client CorrespondentClient
}

func NewCorrespondentResolver(opts CorrespondentResolverOptions) *CorrespondentResolver {
	return newResolver[plclient.Correspondent](&correspondentProvider{opts})
}

func NewMemCorrespondentResolver() *CorrespondentResolver {
	return newMemResolver(func(id int64, name string) plclient.Correspondent {
		return plclient.Correspondent{
			ID:   id,
			Name: name,
		}
	})
}
