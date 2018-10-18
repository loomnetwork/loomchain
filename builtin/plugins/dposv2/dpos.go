package dposv2

import (
	"errors"
	"fmt"
	"os"
	"time"

	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

var (
	decimals                  int64 = 18
	errCandidateNotRegistered       = errors.New("candidate is not registered")
)

type (
	InitRequest                = dtypes.DPOSInitRequestV2
	RegisterCandidateRequest   = dtypes.RegisterCandidateRequestV2
	UnregisterCandidateRequest = dtypes.UnregisterCandidateRequestV2
	ListCandidateRequest       = dtypes.ListCandidateRequestV2
	ListCandidateResponse      = dtypes.ListCandidateResponseV2
	ListValidatorsRequest      = dtypes.ListValidatorsRequestV2
	ListValidatorsResponse     = dtypes.ListValidatorsResponseV2
	VoteRequest                = dtypes.VoteRequestV2
	ElectRequest               = dtypes.ElectRequestV2
	Candidate                  = dtypes.CandidateV2
	Validator                  = types.Validator
	Voter                      = dtypes.VoterV2
	State                      = dtypes.StateV2
	Params                     = dtypes.ParamsV2
)

type DPOS struct {
}

func (c *DPOS) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "dposV2",
		Version: "2.0.0",
	}, nil
}

func (c *DPOS) Init(ctx contract.Context, req *InitRequest) error {
	fmt.Fprintf(os.Stderr, "Init DPOS Params %#v\n", req)
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

	validators := make([]*Validator, len(req.Validators), len(req.Validators))
	for i, val := range req.Validators {
		validators[i] = &Validator{
			PubKey: val.PubKey,
		}
	}

	sortedValidators := sortValidators(validators)
	state := &State{
		Params:           params,
		Validators:       sortedValidators,
		LastElectionTime: ctx.Now().Unix(),
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

	cand := &dtypes.CandidateV2{
		PubKey:  req.PubKey,
		Address: candAddr.MarshalPB(),
	}
	cands.Set(cand)
	return saveCandidateList(ctx, cands)
}

func (c *DPOS) UnregisterCandidate(ctx contract.Context, req *dtypes.UnregisterCandidateRequestV2) error {
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

func (c *DPOS) Vote(ctx contract.Context, req *dtypes.VoteRequestV2) error {
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
		vote = &dtypes.VoteV2{
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

/*
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
*/


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

	witCount := int(params.ValidatorCount)
	if len(results) < witCount {
		witCount = len(results)
	}

	validators := make([]*Validator, witCount, witCount)
	for i, res := range results[:witCount] {
		cand := cands.Get(res.CandidateAddress)
		validators[i] = &Validator{
			PubKey:  cand.PubKey,
			Power:   int64(res.PowerTotal),
		}
	}

	sortedValidators := sortValidators(validators)

	if len(sortedValidators) == 0 {
		return errors.New("there must be at least 1 validator elected")
	}

	if params.ValidatorSalary > 0 {
		// Payout salaries to validators
		coin := &ERC20{
			Context:         ctx,
			ContractAddress: coinAddr,
		}

		salary := sciNot(int64(params.ValidatorSalary), decimals)
		chainID := ctx.Block().ChainID
		for _, wit := range state.Validators {
			witLocalAddr := loom.LocalAddressFromPublicKey(wit.PubKey)
			witAddr := loom.Address{ChainID: chainID, Local: witLocalAddr}
			err = coin.Transfer(witAddr, salary)
			if err != nil {
				return err
			}
		}
	}

	// TODO this will be replaced with Validator updates in `EndBlock`
	// first zero out the current validators
	for _, wit := range state.Validators {
		ctx.SetValidatorPower(wit.PubKey, 0)
	}

	for _, wit := range sortedValidators {
		ctx.SetValidatorPower(wit.PubKey, 100)
	}

	state.Validators = sortedValidators
	state.LastElectionTime = ctx.Now().Unix()
	return saveState(ctx, state)
}

func (c *DPOS) ListValidators(ctx contract.StaticContext, req *ListValidatorsRequest) (*ListValidatorsResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	return &ListValidatorsResponse{
		Validators: state.Validators,
	}, nil
}

// TODO I'd rather remove this
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
	voter *dtypes.VoterV2,
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
