package dpos

import (
	"errors"

	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dpos"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

var (
	decimals                  = 18
	errCandidateNotRegistered = errors.New("candidate is not registered")
)

type (
	InitRequest                = dtypes.InitRequest
	RegisterCandidateRequest   = dtypes.RegisterCandidateRequest
	UnregisterCandidateRequest = dtypes.UnregisterCandidateRequest
	ListWitnessesRequest       = dtypes.ListWitnessesRequest
	ListWitnessesResponse      = dtypes.ListWitnessesResponse
	VoteRequest                = dtypes.VoteRequest
	ProxyVoteRequest           = dtypes.ProxyVoteRequest
	UnproxyVoteRequest         = dtypes.UnproxyVoteRequest
	ElectRequest               = dtypes.ElectRequest
	Candidate                  = dtypes.Candidate
	Witness                    = dtypes.Witness
	Voter                      = dtypes.Voter
	State                      = dtypes.State
	Params                     = dtypes.Params
)

type DPOS struct {
}

func (c *DPOS) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "dpos",
		Version: "1.0.0",
	}, nil
}

func (c *DPOS) Init(ctx contract.Context, req *InitRequest) error {
	params := req.Params

	if params.VoteAllocation == 0 {
		params.VoteAllocation = params.WitnessCount
	}

	if params.CoinContractAddress == nil {
		addr, err := ctx.Resolve("coin")
		if err != nil {
			return err
		}
		params.CoinContractAddress = addr.MarshalPB()
	}

	witnesses := make([]*Witness, len(req.Validators), len(req.Validators))
	for i, val := range req.Validators {
		witnesses[i] = &Witness{
			PubKey: val.PubKey,
		}
	}

	state := &State{
		Params:    params,
		Witnesses: witnesses,
	}

	return saveState(ctx, state)
}

func (c *DPOS) RegisterCandidate(ctx contract.Context, req *RegisterCandidateRequest) error {
	candAddr := ctx.Message().Sender
	cands, err := loadCandidateSet(ctx)
	if err != nil {
		return err
	}

	checkAddr := loom.LocalAddressFromPublicKey(req.PubKey)
	if candAddr.Local.Compare(checkAddr) != 0 {
		return errors.New("public key does not match address")
	}

	cand := &dtypes.Candidate{
		PubKey:  req.PubKey,
		Address: candAddr.MarshalPB(),
	}
	cands.Set(cand)

	return saveCandidateSet(ctx, cands)
}

func (c *DPOS) UnregisterCandidate(ctx contract.Context, req *dtypes.UnregisterCandidateRequest) error {
	candAddr := ctx.Message().Sender
	cands, err := loadCandidateSet(ctx)
	if err != nil {
		return err
	}

	cand := cands.Get(candAddr)
	if cand == nil {
		return errCandidateNotRegistered
	}

	cands.Delete(candAddr)
	// TODO: reallocate votes?
	return saveCandidateSet(ctx, cands)
}

func (c *DPOS) Vote(ctx contract.Context, req *dtypes.VoteRequest) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	params := state.Params

	voterAddr := ctx.Message().Sender
	voter, err := loadVoter(ctx, voterAddr, params.VoteAllocation)
	if err != nil {
		return err
	}

	if int64(voter.Balance) < req.Amount {
		return errors.New("insufficient votes left")
	}

	cands, err := loadCandidateSet(ctx)
	if err != nil {
		return err
	}

	candAddr := loom.UnmarshalAddressPB(req.CandidateAddress)
	cand := cands.Get(candAddr)
	if cand == nil {
		return errCandidateNotRegistered
	}

	votes, err := loadVoteSet(ctx, candAddr)
	if err != nil {
		return err
	}

	vote := votes.Get(voterAddr)
	if vote == nil {
		vote = &dtypes.Vote{
			VoterAddress:     voterAddr.MarshalPB(),
			CandidateAddress: req.CandidateAddress,
		}
	}
	if int64(vote.Amount)+req.Amount < 0 {
		return errors.New("total votes for a candidate must be positive")
	}

	voter.Balance = uint64(int64(voter.Balance) - req.Amount)
	vote.Amount = uint64(int64(vote.Amount) + req.Amount)
	err = saveVoter(ctx, voter)
	if err != nil {
		return err
	}
	votes.Set(vote)
	return saveVoteSet(ctx, candAddr, votes)
}

func (c *DPOS) ProxyVote(ctx contract.Context, req *ProxyVoteRequest) error {
	return nil
}

func (c *DPOS) UnproxyVote(ctx contract.Context, req *UnproxyVoteRequest) error {
	return nil
}

func (c *DPOS) Elect(ctx contract.Context, req *ElectRequest) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	params := state.Params
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)

	cands, err := loadCandidateSet(ctx)
	if err != nil {
		return err
	}

	var fullVotes []*FullVote
	for _, cand := range cands {
		votes, err := loadVoteSet(ctx, loom.UnmarshalAddressPB(cand.Address))
		if err != nil {
			return err
		}
		for _, vote := range votes {
			voter, err := loadVoter(ctx, loom.UnmarshalAddressPB(vote.VoterAddress), params.VoteAllocation)
			if err != nil {
				return err
			}
			votePower, err := calcVotePower(ctx, coinAddr, voter)
			if err != nil {
				return err
			}
			fullVotes = append(fullVotes, &FullVote{
				CandidateAddress: loom.UnmarshalAddressPB(vote.CandidateAddress),
				VoteSize:         vote.Amount,
				Power:            vote.Amount * votePower,
			})
		}
	}

	results, err := runElection(fullVotes)
	if err != nil {
		return err
	}

	witCount := int(params.WitnessCount)
	if len(results) < witCount {
		witCount = len(results)
	}

	witnesses := make([]*Witness, witCount, witCount)
	for i, res := range results[:witCount] {
		cand := cands[addrKey(res.CandidateAddress)]
		witnesses[i] = &Witness{
			PubKey:     cand.PubKey,
			VoteTotal:  res.VoteTotal,
			PowerTotal: res.PowerTotal,
		}
	}

	if len(witnesses) == 0 {
		return errors.New("there must be at least 1 witness elected")
	}

	// first zero out the current validators
	for _, wit := range state.Witnesses {
		ctx.SetValidatorPower(wit.PubKey, 0)
	}

	for _, wit := range witnesses {
		ctx.SetValidatorPower(wit.PubKey, 100)
	}

	state.Witnesses = witnesses
	return saveState(ctx, state)
}

func (c *DPOS) ListWitnesses(ctx contract.StaticContext, req *ListWitnessesRequest) (*ListWitnessesResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	return &ListWitnessesResponse{
		Witnesses: state.Witnesses,
	}, nil
}

func balanceToPower(n *loom.BigUInt) uint64 {
	// TODO: make this configurable
	div := loom.NewBigUIntFromInt(10)
	div.Exp(div, loom.NewBigUIntFromInt(18), nil)
	ret := loom.NewBigUInt(n.Int)
	return ret.Div(ret, div).Uint64()
}

func calcVotePower(
	ctx contract.StaticContext,
	coinAddr loom.Address,
	voter *dtypes.Voter,
) (uint64, error) {
	coin := ERC20Static{
		StaticContext:   ctx,
		ContractAddress: coinAddr,
	}
	total, err := coin.BalanceOf(loom.UnmarshalAddressPB(voter.Address))
	if err != nil {
		return 0, err
	}

	return balanceToPower(total), nil
}

var Contract plugin.Contract = contract.MakePluginContract(&DPOS{})
