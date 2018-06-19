package dpos

import (
	"errors"
	"fmt"
	"time"

	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dpos"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

var (
	decimals                  int64 = 18
	errCandidateNotRegistered       = errors.New("candidate is not registered")
)

type (
	InitRequest                = dtypes.DPOSInitRequest
	RegisterCandidateRequest   = dtypes.RegisterCandidateRequest
	UnregisterCandidateRequest = dtypes.UnregisterCandidateRequest
	ListCandidateRequest       = dtypes.ListCandidateRequest
	ListCandidateResponse      = dtypes.ListCandiateResponse
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
	cands, err := loadCandidateList(ctx)
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
	return saveCandidateList(ctx, cands)
}

func (c *DPOS) UnregisterCandidate(ctx contract.Context, req *dtypes.UnregisterCandidateRequest) error {
	candAddr := ctx.Message().Sender
	cands, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := cands.Get(candAddr)
	if cand == nil {
		return errCandidateNotRegistered
	}

	cands.Delete(candAddr)
	// TODO: reallocate votes?
	return saveCandidateList(ctx, cands)
}

func (c *DPOS) ListCandidates(ctx contract.StaticContext, req *ListCandidateRequest) (*ListCandidateResponse, error) {
	cands, err := loadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	return &ListCandidateResponse{
		Candidates: cands,
	}, nil
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

	cands, err := loadCandidateList(ctx)
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

	cycleLen := time.Duration(params.ElectionCycleLength) * time.Second
	lastTime := time.Unix(state.LastElectionTime, 0)
	if ctx.Now().Sub(lastTime) < cycleLen {
		return fmt.Errorf("must wait at least %d seconds before holding another election", params.ElectionCycleLength)
	}

	cands, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	ctx.Logger().Debug(fmt.Sprintf("candidate list %v", cands))

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

	var resultsPower uint64
	for _, res := range results {
		resultsPower += res.PowerTotal
	}

	staticCoin := &ERC20Static{
		StaticContext:   ctx,
		ContractAddress: coinAddr,
	}
	totalSupply, err := staticCoin.TotalSupply()
	if err != nil {
		return err
	}

	var minPowerReq uint64
	if params.MinPowerFraction > 0 {
		minPowerReq = balanceToPower(totalSupply) / params.MinPowerFraction
	}
	if resultsPower < minPowerReq {
		return errors.New("election did not meet the minimum power required")
	}

	witCount := int(params.WitnessCount)
	if len(results) < witCount {
		witCount = len(results)
	}

	ctx.Logger().Debug(fmt.Sprintf("result list"))
	for i, r := range results {
		ctx.Logger().Debug(fmt.Sprintln(i, r.CandidateAddress.Local.Hex(), r.PowerTotal, r.VoteTotal))
	}

	witnesses := make([]*Witness, witCount, witCount)
	for i, res := range results[:witCount] {
		cand := cands.Get(res.CandidateAddress)
		witnesses[i] = &Witness{
			PubKey:     cand.PubKey,
			VoteTotal:  res.VoteTotal,
			PowerTotal: res.PowerTotal,
		}
	}
	ctx.Logger().Debug(fmt.Sprintf("witness list"))
	for i, r := range witnesses {
		addr := loom.LocalAddressFromPublicKey(r.PubKey)
		ctx.Logger().Debug(fmt.Sprintln(i, addr.Hex(), r.PowerTotal, r.VoteTotal))
	}

	if len(witnesses) == 0 {
		return errors.New("there must be at least 1 witness elected")
	}

	if params.WitnessSalary > 0 {
		// Payout salaries to witnesses
		coin := &ERC20{
			Context:         ctx,
			ContractAddress: coinAddr,
		}

		salary := sciNot(int64(params.WitnessSalary), decimals)
		chainID := ctx.Block().ChainID
		for _, wit := range state.Witnesses {
			witLocalAddr := loom.LocalAddressFromPublicKey(wit.PubKey)
			witAddr := loom.Address{ChainID: chainID, Local: witLocalAddr}
			err = coin.Transfer(witAddr, salary)
			if err != nil {
				return err
			}
		}
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

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

func balanceToPower(n *loom.BigUInt) uint64 {
	// TODO: make this configurable
	div := sciNot(1, decimals)
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
