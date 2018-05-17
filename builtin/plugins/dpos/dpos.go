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
	VoteRequest                = dtypes.VoteRequest
	ProxyVoteRequest           = dtypes.ProxyVoteRequest
	UnproxyVoteRequest         = dtypes.UnproxyVoteRequest
	ElectRequest               = dtypes.ElectRequest
	Candidate                  = dtypes.Candidate
	Params                     = dtypes.Params
)

type DPOS struct {
}

func (c *DPOS) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "coin",
		Version: "1.0.0",
	}, nil
}

func (c *DPOS) Init(ctx contract.Context, req *InitRequest) error {
	params := req.Params

	if params.VoteAllocation == 0 {
		params.VoteAllocation = params.ValidatorCount
	}

	if params.CoinContractAddress == nil {
		addr, err := ctx.Resolve("coin")
		if err != nil {
			return err
		}
		params.CoinContractAddress = addr.MarshalPB()
	}

	state := &dtypes.State{
		Params:     params,
		Validators: req.Validators,
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

	validCount := int(params.ValidatorCount)
	if len(results) < validCount {
		validCount = len(results)
	}

	newValidators := make([]*loom.Validator, validCount, validCount)
	for i, res := range results[:validCount] {
		cand := cands[addrKey(res.CandidateAddress)]
		newValidators[i] = &loom.Validator{
			PubKey: cand.PubKey,
			Power:  100,
		}
	}

	// first zero out the current validators
	for _, val := range state.Validators {
		ctx.SetValidatorPower(val.PubKey, 0)
	}

	for _, val := range newValidators {
		ctx.SetValidatorPower(val.PubKey, val.Power)
	}

	state.Validators = newValidators
	return saveState(ctx, state)
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
