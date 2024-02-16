package objectresolver

import (
	"context"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type UserClient interface {
	ListUsers(context.Context, plclient.ListUsersOptions) ([]plclient.User, *plclient.Response, error)
}

type userProvider struct {
	UserResolverOptions
}

func (userProvider) kind() string {
	return "user"
}

func (p userProvider) listByName(ctx context.Context, name string) ([]plclient.User, error) {
	opts := plclient.ListUsersOptions{}
	opts.Username.EqualsIgnoringCase = &name

	items, _, err := p.Client.ListUsers(ctx, opts)

	return items, err
}

func (p userProvider) create(ctx context.Context, name string) error {
	return ErrCreateUnsupported
}

type UserResolver = Resolver[plclient.User]

type UserResolverOptions struct {
	Client UserClient
}

func NewUserResolver(opts UserResolverOptions) *UserResolver {
	return newResolver[plclient.User](&userProvider{opts})
}

func NewMemUserResolver() *UserResolver {
	return newMemResolver(func(id int64, name string) plclient.User {
		return plclient.User{
			ID:       id,
			Username: name,
		}
	})
}
