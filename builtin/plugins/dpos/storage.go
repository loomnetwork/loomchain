package dpos

import (
	"bytes"
	"fmt"
	"sort"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/dpos"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
)

var (
	economyKey    = []byte("economy")
	stateKey      = []byte("state")
	candidatesKey = []byte("candidates")
)

func addrKey(addr loom.Address) string {
	return string(addr.Bytes())
}

func sortWitnesses(witnesses []*Witness) []*Witness {
	sortedWitnesses := make([]*Witness, len(witnesses))
	copy(sortedWitnesses, witnesses)
	sort.Sort(byPubkey(sortedWitnesses))
	//Its questionable if we should make a copy here or modify the existing slice
	return sortedWitnesses
}

func sortCandidates(cands []*Candidate) []*Candidate {
	sorted := make([]*Candidate, len(cands))
	copy(sorted, cands)
	sort.Sort(byAddress(sorted))
	return sorted
}

func sortVotes(votes []*types.Vote) []*types.Vote {
	sorted := make([]*types.Vote, len(votes))
	copy(sorted, votes)
	sort.Sort(byAddressAndAmount(sorted))
	return sorted
}

type byPubkey []*Witness

func (s byPubkey) Len() int {
	return len(s)
}

func (s byPubkey) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPubkey) Less(i, j int) bool {
	return bytes.Compare(s[i].PubKey, s[j].PubKey) < 0
}

type VoteList []*types.Vote

func (vl VoteList) Get(addr loom.Address) *types.Vote {
	for _, v := range vl {
		addrV := loom.UnmarshalAddressPB(v.VoterAddress)
		if addr.Local.Compare(addrV.Local) == 0 {
			return v
		}
	}
	return nil
}

func (vl *VoteList) Set(vote *types.Vote) {
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

func (s VoteList) String() string {
	var buf = new(bytes.Buffer)
	for _, v := range s {
		addr := loom.UnmarshalAddressPB(v.VoterAddress)
		cand := loom.UnmarshalAddressPB(v.CandidateAddress)
		buf.WriteString(fmt.Sprintf("voter: %s votes %d for candidate: %s,", addr.Local.Hex(), v.Amount, cand.Local.Hex()))
	}
	return buf.String()
}

type byAddressAndAmount []*types.Vote

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

type CandidateList []*types.Candidate

func (c CandidateList) Get(addr loom.Address) *Candidate {
	for _, cand := range c {
		if cand.Address.Local.Compare(addr.Local) == 0 {
			return cand
		}
	}
	return nil
}

func (c CandidateList) String() string {
	var buf = new(bytes.Buffer)
	for _, v := range c {
		addr := loom.UnmarshalAddressPB(v.Address)
		buf.WriteString(fmt.Sprintf("%s, %s\n", v.PubKey, addr))
	}
	return buf.String()
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

type byAddress []*types.Candidate

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
	sort.Sort(byAddress(cl))
	return ctx.Set(candidatesKey, &types.CandidateList{Candidates: cl})
}

func loadCandidateList(ctx contract.StaticContext) (CandidateList, error) {
	var pbcl types.CandidateList
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

func saveVoter(ctx contract.Context, v *types.Voter) error {
	addr := loom.UnmarshalAddressPB(v.Address)
	return ctx.Set(voterKey(addr), v)
}

func loadVoter(ctx contract.Context, addr loom.Address, defaultBalance uint64) (*types.Voter, error) {
	v := types.Voter{
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
	return ctx.Set(voteSetKey(candAddr), &types.VoteList{Votes: sorted})
}

func loadVoteSet(ctx contract.StaticContext, candAddr loom.Address) (VoteList, error) {
	var pbvs types.VoteList
	err := ctx.Get(voteSetKey(candAddr), &pbvs)
	if err == contract.ErrNotFound {
		return VoteList{}, nil
	}
	if err != nil {
		return nil, err
	}

	return pbvs.Votes, nil
}

func saveState(ctx contract.Context, state *types.State) error {
	return ctx.Set(stateKey, state)
}

func loadState(ctx contract.StaticContext) (*types.State, error) {
	var state types.State
	err := ctx.Get(stateKey, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}
