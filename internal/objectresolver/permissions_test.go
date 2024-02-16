package objectresolver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/paperminer/internal/ref"
)

func TestPermissionOptionsApply(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts PermissionOptions
		want map[string]any
	}{
		{name: "defaults"},
		{
			name: "populated",
			opts: PermissionOptions{
				DefaultOwner: ref.Ref[int64](5532),
				DefaultPermissions: &plclient.ObjectPermissions{
					View: plclient.ObjectPermissionPrincipals{
						Users: []int64{25314, 25409},
					},
					Change: plclient.ObjectPermissionPrincipals{
						Groups: []int64{13711, 28015},
					},
				},
			},
			want: map[string]any{
				"owner": plclient.Int64(5532),
				"set_permissions": &plclient.ObjectPermissions{
					View: plclient.ObjectPermissionPrincipals{
						Users: []int64{25314, 25409},
					},
					Change: plclient.ObjectPermissionPrincipals{
						Groups: []int64{13711, 28015},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := plclient.NewTagFields()

			tc.opts.apply(fields)

			if diff := cmp.Diff(tc.want, fields.AsMap(), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("apply() diff (-want +got):\n%s", diff)
			}
		})
	}
}
