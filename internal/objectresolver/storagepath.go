package objectresolver

import (
	"context"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type StoragePathClient interface {
	ListStoragePaths(context.Context, plclient.ListStoragePathsOptions) ([]plclient.StoragePath, *plclient.Response, error)
	CreateStoragePath(context.Context, *plclient.StoragePathFields) (*plclient.StoragePath, *plclient.Response, error)
}

type storagePathProvider struct {
	StoragePathResolverOptions
}

func (storagePathProvider) kind() string {
	return "storagePath"
}

func (p *storagePathProvider) create(ctx context.Context, name string) error {
	return ErrCreateUnsupported
}

func (p *storagePathProvider) listByName(ctx context.Context, name string) ([]plclient.StoragePath, error) {
	opts := plclient.ListStoragePathsOptions{}
	opts.Name.EqualsIgnoringCase = &name

	items, _, err := p.Client.ListStoragePaths(ctx, opts)

	return items, err
}

type StoragePathResolver = Resolver[plclient.StoragePath]

type StoragePathResolverOptions struct {
	PermissionOptions

	Client StoragePathClient
}

func NewStoragePathResolver(opts StoragePathResolverOptions) *StoragePathResolver {
	return newResolver[plclient.StoragePath](&storagePathProvider{opts})
}

func NewMemStoragePathResolver() *StoragePathResolver {
	return newMemResolver(func(id int64, name string) plclient.StoragePath {
		return plclient.StoragePath{
			ID:   id,
			Name: name,
		}
	})
}
