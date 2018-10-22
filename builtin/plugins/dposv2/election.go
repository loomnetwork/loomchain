package dposv2

import (
	loom "github.com/loomnetwork/go-loom"
)

type DelegationResult struct {
	ValidatorAddress loom.Address
	DelegationTotal  loom.BigUInt
}

type byDelegationTotal []*DelegationResult

func (s byDelegationTotal) Len() int {
	return len(s)
}

func (s byDelegationTotal) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byDelegationTotal) Less(i, j int) bool {
	diff := int64(s[i].DelegationTotal.Cmp(&s[j].DelegationTotal))
	if diff == 0 {
		// make sure output is deterministic if power is equal
		diff = int64(s[i].ValidatorAddress.Compare(s[j].ValidatorAddress))
	}

	return diff < 0
}
