package objectresolver

import (
	"context"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type DocumentTypeClient interface {
	ListDocumentTypes(context.Context, plclient.ListDocumentTypesOptions) ([]plclient.DocumentType, *plclient.Response, error)
	CreateDocumentType(context.Context, *plclient.DocumentTypeFields) (*plclient.DocumentType, *plclient.Response, error)
}

type documentTypeProvider struct {
	DocumentTypeResolverOptions
}

func (documentTypeProvider) kind() string {
	return "document type"
}

func (p *documentTypeProvider) create(ctx context.Context, name string) error {
	fields := plclient.NewDocumentTypeFields().
		SetName(name).
		SetMatchingAlgorithm(plclient.MatchNone)

	p.PermissionOptions.apply(fields)

	_, _, err := p.Client.CreateDocumentType(ctx, fields)

	return err
}

func (p *documentTypeProvider) listByName(ctx context.Context, name string) ([]plclient.DocumentType, error) {
	opts := plclient.ListDocumentTypesOptions{}
	opts.Name.EqualsIgnoringCase = &name

	items, _, err := p.Client.ListDocumentTypes(ctx, opts)

	return items, err
}

type DocumentTypeResolver = Resolver[plclient.DocumentType]

type DocumentTypeResolverOptions struct {
	PermissionOptions

	Client DocumentTypeClient
}

func NewDocumentTypeResolver(opts DocumentTypeResolverOptions) *DocumentTypeResolver {
	return newResolver[plclient.DocumentType](&documentTypeProvider{opts})
}

func NewMemDocumentTypeResolver() *DocumentTypeResolver {
	return newMemResolver(func(id int64, name string) plclient.DocumentType {
		return plclient.DocumentType{
			ID:   id,
			Name: name,
		}
	})
}
