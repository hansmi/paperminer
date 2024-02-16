package objectresolver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/paperhooks/pkg/client"
	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type fakeObjectResolverClient struct {
	err error

	receivedDocumentTypeFields *plclient.DocumentTypeFields
}

func (c *fakeObjectResolverClient) GetCurrentUser(context.Context) (*plclient.User, *plclient.Response, error) {
	return &plclient.User{
		ID:       123,
		Username: "currentuser",
	}, nil, nil
}

func (c *fakeObjectResolverClient) ListUsers(context.Context, plclient.ListUsersOptions) ([]plclient.User, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) CreateUser(context.Context, *plclient.UserFields) (*plclient.User, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) ListGroups(context.Context, plclient.ListGroupsOptions) ([]plclient.Group, *plclient.Response, error) {
	return []plclient.Group{
		{
			ID:   19092,
			Name: "mygroup",
		},
	}, nil, nil
}

func (c *fakeObjectResolverClient) CreateGroup(context.Context, *plclient.GroupFields) (*plclient.Group, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) ListTags(context.Context, plclient.ListTagsOptions) ([]plclient.Tag, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) CreateTag(context.Context, *plclient.TagFields) (*plclient.Tag, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) ListCorrespondents(context.Context, plclient.ListCorrespondentsOptions) ([]plclient.Correspondent, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) CreateCorrespondent(context.Context, *plclient.CorrespondentFields) (*plclient.Correspondent, *plclient.Response, error) {
	return nil, nil, c.err
}

func (c *fakeObjectResolverClient) ListDocumentTypes(context.Context, plclient.ListDocumentTypesOptions) ([]plclient.DocumentType, *plclient.Response, error) {
	var result []plclient.DocumentType

	if c.receivedDocumentTypeFields != nil {
		result = append(result, plclient.DocumentType{
			ID:   25128,
			Name: "abc",
		})
	}

	return result, nil, nil
}

func (c *fakeObjectResolverClient) CreateDocumentType(_ context.Context, fields *plclient.DocumentTypeFields) (*plclient.DocumentType, *plclient.Response, error) {
	c.receivedDocumentTypeFields = fields

	return &plclient.DocumentType{
		ID:   789,
		Name: "created type",
	}, nil, nil
}

func (c *fakeObjectResolverClient) ListStoragePaths(context.Context, plclient.ListStoragePathsOptions) ([]plclient.StoragePath, *plclient.Response, error) {
	return []plclient.StoragePath{}, nil, nil
}

func (c *fakeObjectResolverClient) CreateStoragePath(context.Context, *plclient.StoragePathFields) (*plclient.StoragePath, *plclient.Response, error) {
	return nil, nil, c.err
}

func TestObjectResolvers(t *testing.T) {
	errTest := errors.New("test error")

	for _, tc := range []struct {
		name                   string
		defaultPermissions     NamedObjectPermissions
		wantErr                error
		wantDocumentTypeFields map[string]any
	}{
		{
			name: "empty",
			wantDocumentTypeFields: map[string]any{
				"name":            "abc",
				"owner":           plclient.Int64(123),
				"set_permissions": &client.ObjectPermissions{},
			},
		},
		{
			name: "owner",
			defaultPermissions: NamedObjectPermissions{
				Owner: "foo",
			},
			wantErr: errTest,
		},
		{
			name: "permissions",
			defaultPermissions: NamedObjectPermissions{
				Change: NamedObjectPermissionPrincipals{
					Groups: []string{"mygroup"},
				},
			},
			wantDocumentTypeFields: map[string]any{
				"name":  "abc",
				"owner": plclient.Int64(123),
				"set_permissions": &client.ObjectPermissions{
					Change: client.ObjectPermissionPrincipals{
						Groups: []int64{19092},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			client := &fakeObjectResolverClient{
				err: errTest,
			}

			got, err := NewObjectResolvers(ctx, client, tc.defaultPermissions)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				wantDocumentType := plclient.DocumentType{
					ID:   25128,
					Name: "abc",
				}

				if gotDocumentType, err := got.DocumentType.GetOrCreateByName(ctx, "abc"); err != nil {
					t.Errorf("Getting document type failed: %v", err)
				} else if diff := cmp.Diff(wantDocumentType, gotDocumentType, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Document type diff (-want +got):\n%s", diff)
				} else if diff := cmp.Diff(tc.wantDocumentTypeFields, client.receivedDocumentTypeFields.AsMap(), cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Document type creation diff (-want +got):\n%s", diff)
				}

				if _, err := got.StoragePath.GetOrCreateByName(ctx, "xyz"); !errors.Is(err, ErrCreateUnsupported) {
					t.Errorf("Creating storage path didn't fail with %v: %v", ErrCreateUnsupported, err)
				}
			}
		})
	}
}
