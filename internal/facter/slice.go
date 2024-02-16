package facter

import (
	"fmt"

	"github.com/hansmi/paperminer"
)

type FactsSlice []*paperminer.Facts

func (s FactsSlice) Best() (*paperminer.Facts, error) {
	switch len(s) {
	case 0:
		return nil, nil

	case 1:
		return s[0], nil
	}

	// TODO: Add some sort of ranking and select the best facts if there are
	// multiple candidates. Maybe over all document variants (archived,
	// original). Facts can also be given a number as a "relative weight".

	return nil, fmt.Errorf("unable to select among %d facts", len(s))
}
