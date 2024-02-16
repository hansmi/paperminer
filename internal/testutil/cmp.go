package testutil

import "github.com/google/go-cmp/cmp/cmpopts"

var CmpSortInt64Slices = cmpopts.SortSlices(func(a, b int64) bool {
	return a < b
})
