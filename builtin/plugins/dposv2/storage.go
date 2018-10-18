package dposv2

import (
	"bytes"
	"sort"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/types"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
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

func sortVotes(votes []*dtypes.VoteV2) []*dtypes.VoteV2 {
	sort.Sort(byAddressAndAmount(votes))
	return votes
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

type VoteList []*dtypes.VoteV2

func (vl VoteList) Get(addr loom.Address) *dtypes.VoteV2 {
	for _, v := range vl {
		addrV := loom.UnmarshalAddressPB(v.VoterAddress)
		if addr.Local.Compare(addrV.Local) == 0 {
			return v
		}
	}
	return nil
}

func (vl *VoteList) Set(vote *dtypes.VoteV2) {
	addr := loom.UnmarshalAddressPB(vote.VoterAddress)
	found := false
	for _, v := range *vl {
		addrV := loom.UnmarshalAddressPB(v.VoterAddress)
		if addr.Local.Compare(addrV.Local) == 0 {
			v = vote
			found = true
			break
		}
	}
	if !found {
		*vl = append(*vl, vote)
	}
}

type DelegationList []*dtypes.DelegationV2

func (dl DelegationList) Get(validator types.Address, delegator types.Address) *Delegation {
	for _, delegation := range dl {
		// TODO there should be a compare defined for all address types
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
	// TODO why is the typing here questionable?
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


type byAddressAndAmount []*dtypes.VoteV2

func (s byAddressAndAmount) Len() int {
	return len(s)
}

func (s byAddressAndAmount) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byAddressAndAmount) Less(i, j int) bool {
	vaddr1 := loom.UnmarshalAddressPB(s[i].VoterAddress)
	vaddr2 := loom.UnmarshalAddressPB(s[j].VoterAddress)
	diff := vaddr1.Local.Compare(vaddr2.Local)
	if diff == 0 {
		caddr1 := loom.UnmarshalAddressPB(s[i].CandidateAddress)
		caddr2 := loom.UnmarshalAddressPB(s[j].CandidateAddress)
		diff = caddr1.Local.Compare(caddr2.Local)

		if diff == 0 {
			return s[i].Amount < s[j].Amount
		}
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

func voterKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("voter"), addr.Bytes())
}

func saveVoter(ctx contract.Context, v *dtypes.VoterV2) error {
	addr := loom.UnmarshalAddressPB(v.Address)
	return ctx.Set(voterKey(addr), v)
}

func loadVoter(ctx contract.Context, addr loom.Address, defaultBalance uint64) (*dtypes.VoterV2, error) {
	v := dtypes.VoterV2{
		Address: addr.MarshalPB(),
		Balance: defaultBalance,
	}
	err := ctx.Get(voterKey(addr), &v)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}

	return &v, nil
}

func voteSetKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("votes"), addr.Bytes())
}

func saveVoteSet(ctx contract.Context, candAddr loom.Address, vs VoteList) error {
	sorted := sortVotes(vs)
	return ctx.Set(voteSetKey(candAddr), &dtypes.VoteListV2{Votes: sorted})
}

func loadVoteSet(ctx contract.StaticContext, candAddr loom.Address) (VoteList, error) {
	var pbvs dtypes.VoteListV2
	err := ctx.Get(voteSetKey(candAddr), &pbvs)
	if err == contract.ErrNotFound {
		return VoteList{}, nil
	}
	if err != nil {
		return nil, err
	}

	return pbvs.Votes, nil
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
