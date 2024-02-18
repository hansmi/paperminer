package cataloger

import (
	"context"
	"fmt"
	"time"

	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/objectresolver"
	"github.com/hansmi/paperminer/internal/ref"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func normalizeIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	ids = slices.Clone(ids)

	slices.Sort(ids)

	return slices.Compact(ids)
}

type patchBuilder struct {
	resolvers *objectresolver.ObjectResolvers
	doc       *plclient.Document

	created       *time.Time
	title         *string
	correspondent **int64
	documentType  **int64
	storagePath   **int64

	tags map[int64]struct{}
}

func newPatchBuilder(resolvers *objectresolver.ObjectResolvers, doc *plclient.Document) *patchBuilder {
	b := &patchBuilder{
		resolvers: resolvers,
		doc:       doc,
		tags:      map[int64]struct{}{},
	}

	for _, tag := range doc.Tags {
		b.tags[tag] = struct{}{}
	}

	return b
}

func (b *patchBuilder) setTag(id int64) {
	b.tags[id] = struct{}{}
}

func (b *patchBuilder) unsetTag(id int64) {
	if _, ok := b.tags[id]; ok {
		delete(b.tags, id)
	}
}

func setObjectFact[T any](ctx context.Context,
	dest ***int64,
	resolver *objectresolver.Resolver[T],
	name *string,
	getID func(T) int64,
) error {
	if name == nil {
		// Not configured
		*dest = nil
	} else if *name == "" {
		// Unset
		*dest = ref.Ref[*int64](nil)
	} else if obj, err := resolver.GetOrCreateByName(ctx, *name); err != nil {
		return err
	} else {
		id := getID(obj)

		*dest = ref.Ref(&id)
	}

	return nil
}

func (b *patchBuilder) setFacts(ctx context.Context, facts *paperminer.Facts) error {
	b.created = facts.Created
	b.title = facts.Title

	if err := setObjectFact(ctx, &b.correspondent, b.resolvers.Correspondent, facts.Correspondent, func(obj plclient.Correspondent) int64 {
		return obj.ID
	}); err != nil {
		return fmt.Errorf("correspondent: %w", err)
	}

	if err := setObjectFact(ctx, &b.documentType, b.resolvers.DocumentType, facts.DocumentType, func(obj plclient.DocumentType) int64 {
		return obj.ID
	}); err != nil {
		return fmt.Errorf("document type: %w", err)
	}

	if err := setObjectFact(ctx, &b.storagePath, b.resolvers.StoragePath, facts.StoragePath, func(obj plclient.StoragePath) int64 {
		return obj.ID
	}); err != nil {
		return fmt.Errorf("storage path: %w", err)
	}

	for _, i := range []struct {
		names   []string
		resolve func(context.Context, string) (plclient.Tag, error)
		apply   func(int64)
	}{
		{facts.SetTags, b.resolvers.Tag.GetOrCreateByName, b.setTag},
		{facts.UnsetTags, b.resolvers.Tag.GetByName, b.unsetTag},
	} {
		for _, name := range i.names {
			if tag, err := i.resolve(ctx, name); err != nil {
				return fmt.Errorf("tag: %w", err)
			} else {
				i.apply(tag.ID)
			}
		}
	}

	return nil
}

func patchOptionalValue[T string | *int64](
	patch *plclient.DocumentFields,
	set func(*plclient.DocumentFields, T) *plclient.DocumentFields,
	value *T,
	current T,
) *plclient.DocumentFields {
	if !(value == nil || *value == current) {
		patch = set(patch, *value)
	}

	return patch
}

func patchOptionalTimeValue(
	patch *plclient.DocumentFields,
	set func(*plclient.DocumentFields, time.Time) *plclient.DocumentFields,
	value *time.Time,
	current time.Time,
) *plclient.DocumentFields {
	if !(value == nil || (*value).Equal(current)) {
		patch = set(patch, *value)
	}

	return patch
}

// Build returns changed fields.
func (b *patchBuilder) build() *plclient.DocumentFields {
	patch := plclient.NewDocumentFields()

	patch = patchOptionalTimeValue(patch,
		(*plclient.DocumentFields).SetCreated, b.created,
		b.doc.Created)

	patch = patchOptionalValue(patch,
		(*plclient.DocumentFields).SetTitle, b.title,
		b.doc.Title)

	patch = patchOptionalValue(patch,
		(*plclient.DocumentFields).SetCorrespondent, b.correspondent,
		b.doc.Correspondent)

	patch = patchOptionalValue(patch,
		(*plclient.DocumentFields).SetDocumentType, b.documentType,
		b.doc.DocumentType)

	patch = patchOptionalValue(patch,
		(*plclient.DocumentFields).SetStoragePath, b.storagePath,
		b.doc.StoragePath)

	if tags, origTags := normalizeIDs(maps.Keys(b.tags)), normalizeIDs(b.doc.Tags); !slices.Equal(tags, origTags) {
		patch = patch.SetTags(tags)
	}

	return patch
}
