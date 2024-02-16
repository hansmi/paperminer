package objectresolver

import "errors"

var (
	ErrAmbiguous         = errors.New("ambiguous object list result")
	ErrCreateUnsupported = errors.New("object creation not supported")
	ErrNotFound          = errors.New("object not found")
)
