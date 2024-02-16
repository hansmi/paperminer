package document

import (
	"fmt"
)

//go:generate stringer -type=Variant -linecomment -output=variant_string.go
type Variant int

var _ fmt.Stringer = (*Variant)(nil)

const (
	Archived Variant = iota // archived
	Original                // original
)
