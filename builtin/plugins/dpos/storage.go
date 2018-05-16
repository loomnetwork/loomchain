package main

import (
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

type CandidateSet map[string]*types.Candidate

func (cs CandidateSet) Get(addr loom.Address) *types.Candidate {
	return cs[addrKey(addr)]
}

func (cs CandidateSet) Set(cand *types.Candidate) {
	cs[addrKey(loom.UnmarshalAddressPB(cand.Address))] = cand
}

func (cs CandidateSet) Delete(addr loom.Address) {
	delete(cs, addrKey(addr))
}

type VoteSet map[string]*types.Vote

func (vs VoteSet) Get(addr loom.Address) *types.Vote {
	return vs[addrKey(addr)]
}

func (vs VoteSet) Set(vote *types.Vote) {
	vs[addrKey(loom.UnmarshalAddressPB(vote.VoterAddress))] = vote
}

func saveCandidateSet(ctx contract.Context, cs CandidateSet) error {
	return ctx.Set(candidatesKey, &types.CandidateSet{Candidates: cs})
}

func loadCandidateSet(ctx contract.StaticContext) (CandidateSet, error) {
	var pbcs types.CandidateSet
	err := ctx.Get(candidatesKey, &pbcs)
	if err == contract.ErrNotFound {
		return make(CandidateSet), nil
	}
	if err != nil {
		return nil, err
	}

	return pbcs.Candidates, nil
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

func saveVoteSet(ctx contract.Context, candAddr loom.Address, vs VoteSet) error {
	return ctx.Set(voteSetKey(candAddr), &types.VoteSet{Votes: vs})
}

func loadVoteSet(ctx contract.StaticContext, candAddr loom.Address) (VoteSet, error) {
	var pbvs types.VoteSet
	err := ctx.Get(voteSetKey(candAddr), &pbvs)
	if err == contract.ErrNotFound {
		return make(VoteSet), nil
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
