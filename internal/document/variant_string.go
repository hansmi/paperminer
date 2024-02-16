// Code generated by "stringer -type=Variant -linecomment -output=variant_string.go"; DO NOT EDIT.

package document

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Archived-0]
	_ = x[Original-1]
}

const _Variant_name = "archivedoriginal"

var _Variant_index = [...]uint8{0, 8, 16}

func (i Variant) String() string {
	if i < 0 || i >= Variant(len(_Variant_index)-1) {
		return "Variant(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Variant_name[_Variant_index[i]:_Variant_index[i+1]]
}
