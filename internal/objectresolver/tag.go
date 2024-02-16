package objectresolver

import (
	"context"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type TagClient interface {
	ListTags(context.Context, plclient.ListTagsOptions) ([]plclient.Tag, *plclient.Response, error)
	CreateTag(context.Context, *plclient.TagFields) (*plclient.Tag, *plclient.Response, error)
}

type tagProvider struct {
	TagResolverOptions
}

func (tagProvider) kind() string {
	return "tag"
}

func (p *tagProvider) create(ctx context.Context, name string) error {
	fields := plclient.NewTagFields().SetName(name)

	p.PermissionOptions.apply(fields)

	_, _, err := p.Client.CreateTag(ctx, fields)

	return err
}

func (p *tagProvider) listByName(ctx context.Context, name string) ([]plclient.Tag, error) {
	opts := plclient.ListTagsOptions{}
	opts.Name.EqualsIgnoringCase = &name

	items, _, err := p.Client.ListTags(ctx, opts)

	return items, err
}

type TagResolver = Resolver[plclient.Tag]

type TagResolverOptions struct {
	PermissionOptions

	Client TagClient
}

func NewTagResolver(opts TagResolverOptions) *TagResolver {
	return newResolver[plclient.Tag](&tagProvider{opts})
}

func NewMemTagResolver() *TagResolver {
	return newMemResolver(func(id int64, name string) plclient.Tag {
		return plclient.Tag{
			ID:   id,
			Name: name,
		}
	})
}
