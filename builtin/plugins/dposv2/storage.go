package dposv2

import (
	"bytes"
	"sort"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/types"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

var (
	stateKey       = []byte("state")
	candidatesKey  = []byte("candidates")
	delegationsKey = []byte("delegation")
)

func addrKey(addr loom.Address) string {
	return string(addr.Bytes())
}

func sortValidators(validators []*Validator) []*Validator {
	sort.Sort(byPubkey(validators))
	return validators
}

func sortDelegations(delegations []*Delegation) []*Delegation {
	sort.Sort(byValidatorAndDelegator(delegations))
	return delegations
}

func sortCandidates(cands []*Candidate) []*Candidate {
	sort.Sort(byAddress(cands))
	return cands
}

type byPubkey []*Validator

func (s byPubkey) Len() int {
	return len(s)
}

func (s byPubkey) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPubkey) Less(i, j int) bool {
	return bytes.Compare(s[i].PubKey, s[j].PubKey) < 0
}

type DelegationList []*dtypes.DelegationV2

func (dl DelegationList) Get(validator types.Address, delegator types.Address) *Delegation {
	for _, delegation := range dl {
		// TODO shouldn't I just convert to loom.Address and use its compare?
		if delegation.Validator.Local.Compare(validator.Local) == 0 && delegation.Delegator.Local.Compare(delegator.Local) == 0 {
			return delegation
		}
	}
	return nil
}

func (dl *DelegationList) Set(delegation *Delegation) {
	pastvalue := dl.Get(*delegation.Validator, *delegation.Delegator)
	if pastvalue == nil {
		*dl = append(*dl, delegation)
	} else {
		pastvalue.Amount = delegation.Amount
		pastvalue.Height = delegation.Height
	}
}

// func (c *DelegationList) Delete(validator loom.Address, delegator loom.Address) {

func saveDelegationList(ctx contract.Context, dl DelegationList) error {
	sorted := sortDelegations(dl)
	return ctx.Set(delegationsKey, &dtypes.DelegationListV2{Delegations: sorted})
}

func loadDelegationList(ctx contract.StaticContext) (DelegationList, error) {
	var pbcl dtypes.DelegationListV2
	err := ctx.Get(delegationsKey, &pbcl)
	if err == contract.ErrNotFound {
		return DelegationList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Delegations, nil
}

type byValidatorAndDelegator []*dtypes.DelegationV2

func (s byValidatorAndDelegator) Len() int {
	return len(s)
}

func (s byValidatorAndDelegator) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byValidatorAndDelegator) Less(i, j int) bool {
	vAddr1 := loom.UnmarshalAddressPB(s[i].Validator)
	vAddr2 := loom.UnmarshalAddressPB(s[j].Validator)
	diff := vAddr1.Compare(vAddr2)

	if diff == 0 {
		dAddr1 := loom.UnmarshalAddressPB(s[i].Delegator)
		dAddr2 := loom.UnmarshalAddressPB(s[j].Delegator)
		diff = dAddr1.Compare(dAddr2)
	}

	return diff < 0
}

type CandidateList []*dtypes.CandidateV2

func (c CandidateList) Get(addr loom.Address) *Candidate {
	for _, cand := range c {
		if cand.Address.Local.Compare(addr.Local) == 0 {
			return cand
		}
	}
	return nil
}

func (c *CandidateList) Set(cand *Candidate) {
	found := false
	candAddr := loom.UnmarshalAddressPB(cand.Address)
	for _, candidate := range *c {
		addr := loom.UnmarshalAddressPB(candidate.Address)
		if candAddr.Local.Compare(addr.Local) == 0 {
			candidate = cand
			found = true
			break
		}
	}
	if !found {
		*c = append(*c, cand)
	}
}

func (c *CandidateList) Delete(addr loom.Address) {
	var newcl CandidateList
	for _, cand := range *c {
		candAddr := loom.UnmarshalAddressPB(cand.Address)
		addr := loom.UnmarshalAddressPB(cand.Address)
		if candAddr.Local.Compare(addr.Local) != 0 {
			newcl = append(newcl, cand)
		}
	}
	*c = newcl
}

type byAddress []*dtypes.CandidateV2

func (s byAddress) Len() int {
	return len(s)
}

func (s byAddress) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byAddress) Less(i, j int) bool {
	vaddr1 := loom.UnmarshalAddressPB(s[i].Address)
	vaddr2 := loom.UnmarshalAddressPB(s[j].Address)
	diff := vaddr1.Local.Compare(vaddr2.Local)
	return diff < 0
}

func saveCandidateList(ctx contract.Context, cl CandidateList) error {
	sorted := sortCandidates(cl)
	return ctx.Set(candidatesKey, &dtypes.CandidateListV2{Candidates: sorted})
}

func loadCandidateList(ctx contract.StaticContext) (CandidateList, error) {
	var pbcl dtypes.CandidateListV2
	err := ctx.Get(candidatesKey, &pbcl)
	if err == contract.ErrNotFound {
		return CandidateList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Candidates, nil
}

func saveState(ctx contract.Context, state *dtypes.StateV2) error {
	return ctx.Set(stateKey, state)
}

func loadState(ctx contract.StaticContext) (*dtypes.StateV2, error) {
	var state dtypes.StateV2
	err := ctx.Get(stateKey, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

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

	return diff > 0
}
