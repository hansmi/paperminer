package objectresolver

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/kpflagvalue"
)

type ResolveOwnerClient interface {
	GetCurrentUser(context.Context) (*plclient.User, *plclient.Response, error)
}

type NamedObjectPermissionPrincipals struct {
	Users  []string
	Groups []string
}

type NamedObjectPermissions struct {
	Owner  string
	View   NamedObjectPermissionPrincipals
	Change NamedObjectPermissionPrincipals
}

func (p *NamedObjectPermissions) resolveOwner(ctx context.Context, cl ResolveOwnerClient, userResolver *UserResolver) (*int64, error) {
	if p.Owner == "" {
		user, _, err := cl.GetCurrentUser(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting current user: %w", err)
		}

		return &user.ID, nil
	}

	user, err := userResolver.GetByName(ctx, p.Owner)
	if err != nil {
		return nil, fmt.Errorf("getting owner user %q: %w", p.Owner, err)
	}

	return &user.ID, nil
}

func (p *NamedObjectPermissions) resolvePermissions(ctx context.Context, userResolver *UserResolver, groupResolver *GroupResolver) (*plclient.ObjectPermissions, error) {
	result := &plclient.ObjectPermissions{}

	for _, i := range []struct {
		dst *plclient.ObjectPermissionPrincipals
		src NamedObjectPermissionPrincipals
	}{
		{&result.View, p.View},
		{&result.Change, p.Change},
	} {
		var users []int64
		var groups []int64

		for _, name := range i.src.Users {
			user, err := userResolver.GetByName(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("getting user %q: %w", name, err)
			}

			users = append(users, user.ID)
		}

		for _, name := range i.src.Groups {
			group, err := groupResolver.GetByName(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("getting group %q: %w", name, err)
			}

			groups = append(groups, group.ID)
		}

		slices.Sort(users)
		slices.Sort(groups)

		i.dst.Users = slices.Compact(users)
		i.dst.Groups = slices.Compact(groups)
	}

	return result, nil
}

func (p *NamedObjectPermissions) RegisterFlags(app *kingpin.Application) {
	app.Flag("object_default_owner_name", "Owner for newly created objects (defaults to authenticated user).").
		PlaceHolder("USER").
		StringVar(&p.Owner)

	defaultPermFlag := func(perm, kind string, target *[]string) {
		kpflagvalue.CommaSeparatedStringsVar(
			app.Flag(fmt.Sprintf("object_default_%s_%s", perm, kind),
				fmt.Sprintf("%s granted %s permission on newly created objects (comma-separated).", strings.Title(kind), perm)).
				PlaceHolder(strings.ToUpper(kind)),
			target)
	}

	defaultPermFlag("view", "users", &p.View.Users)
	defaultPermFlag("view", "groups", &p.View.Groups)
	defaultPermFlag("change", "users", &p.Change.Users)
	defaultPermFlag("change", "groups", &p.Change.Groups)
}

type ObjectResolverClient interface {
	ResolveOwnerClient
	UserClient
	GroupClient
	TagClient
	CorrespondentClient
	DocumentTypeClient
	StoragePathClient
}

type ObjectResolvers struct {
	User          *UserResolver
	Group         *GroupResolver
	Tag           *TagResolver
	Correspondent *CorrespondentResolver
	DocumentType  *DocumentTypeResolver
	StoragePath   *StoragePathResolver
}

func NewObjectResolvers(ctx context.Context, cl ObjectResolverClient, defaultPerm NamedObjectPermissions) (*ObjectResolvers, error) {
	userResolver := NewUserResolver(UserResolverOptions{
		Client: cl,
	})

	groupResolver := NewGroupResolver(GroupResolverOptions{
		Client: cl,
	})

	var permOpts PermissionOptions

	if owner, err := defaultPerm.resolveOwner(ctx, cl, userResolver); err != nil {
		return nil, err
	} else {
		permOpts.DefaultOwner = owner
	}

	if perm, err := defaultPerm.resolvePermissions(ctx, userResolver, groupResolver); err != nil {
		return nil, err
	} else {
		permOpts.DefaultPermissions = perm
	}

	return &ObjectResolvers{
		User:  userResolver,
		Group: groupResolver,
		Tag: NewTagResolver(TagResolverOptions{
			PermissionOptions: permOpts,
			Client:            cl,
		}),
		Correspondent: NewCorrespondentResolver(CorrespondentResolverOptions{
			PermissionOptions: permOpts,
			Client:            cl,
		}),
		DocumentType: NewDocumentTypeResolver(DocumentTypeResolverOptions{
			PermissionOptions: permOpts,
			Client:            cl,
		}),
		StoragePath: NewStoragePathResolver(StoragePathResolverOptions{
			PermissionOptions: permOpts,
			Client:            cl,
		}),
	}, nil
}

func NewMemObjectResolvers() *ObjectResolvers {
	return &ObjectResolvers{
		User:          NewMemUserResolver(),
		Group:         NewMemGroupResolver(),
		Tag:           NewMemTagResolver(),
		Correspondent: NewMemCorrespondentResolver(),
		DocumentType:  NewMemDocumentTypeResolver(),
		StoragePath:   NewMemStoragePathResolver(),
	}
}
