package objectresolver

import (
	"context"
	"errors"
	"testing"

	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/ref"
)

type fakeTagClient struct {
	tag *plclient.Tag
}

func (c *fakeTagClient) ListTags(context.Context, plclient.ListTagsOptions) ([]plclient.Tag, *plclient.Response, error) {
	if c.tag == nil {
		return nil, nil, nil
	}

	return []plclient.Tag{*c.tag}, nil, nil
}

func (c *fakeTagClient) CreateTag(context.Context, *plclient.TagFields) (*plclient.Tag, *plclient.Response, error) {
	if c.tag == nil {
		return &plclient.Tag{}, nil, nil
	}

	return ref.Ref(*c.tag), nil, nil
}

func TestTagResolver(t *testing.T) {
	ctx := context.Background()

	cl := &fakeTagClient{}
	r := NewTagResolver(TagResolverOptions{
		Client: cl,
	})

	for _, name := range []string{"", "missing"} {
		if got, err := r.GetByName(ctx, name); !errors.Is(err, ErrNotFound) {
			t.Errorf("GetByName(%q) returned unexpected error (%#v): %v", name, got, err)
		}

		if got, err := r.GetOrCreateByName(ctx, name); !errors.Is(err, ErrNotFound) {
			t.Errorf("GetOrCreateByName(%q) returned unexpected error (%#v): %v", name, got, err)
		}
	}

	cl.tag = &plclient.Tag{
		ID:   123,
		Name: "test",
	}

	if got, err := r.GetByName(ctx, "ignored in test"); err != nil {
		t.Errorf("GetByName() failed: %v", err)
	} else if !(got.ID == 123 && got.Name == "test") {
		t.Errorf("GetByName() returned unexpected %#v", got)
	}
}

func TestMemTagResolver(t *testing.T) {
	validateMemResolver(t, NewMemTagResolver())
}
