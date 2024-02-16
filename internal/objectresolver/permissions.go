package objectresolver

import (
	"reflect"

	plclient "github.com/hansmi/paperhooks/pkg/client"
)

type PermissionOptions struct {
	DefaultOwner       *int64
	DefaultPermissions *plclient.ObjectPermissions
}

func (o PermissionOptions) apply(fields any) {
	f := reflect.ValueOf(fields)

	if o.DefaultOwner != nil {
		f.MethodByName("SetOwner").Call([]reflect.Value{
			reflect.ValueOf(o.DefaultOwner),
		})
	}

	if o.DefaultPermissions != nil {
		f.MethodByName("SetSetPermissions").Call([]reflect.Value{
			reflect.ValueOf(o.DefaultPermissions),
		})
	}
}
