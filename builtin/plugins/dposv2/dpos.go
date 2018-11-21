package dposv2

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"

	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
)

var (
	decimals                  int64 = 18
	errCandidateNotRegistered  = errors.New("candidate is not registered")
	errValidatorNotFound       = errors.New("validator not found")
	errDistributionNotFound       = errors.New("distribution not found")
)

type (
	InitRequest                = dtypes.DPOSInitRequestV2
	DelegateRequest            = dtypes.DelegateRequestV2
	UnbondRequest              = dtypes.UnbondRequestV2
	ClaimDistributionRequest   = dtypes.ClaimDistributionRequestV2
	ClaimDistributionResponse  = dtypes.ClaimDistributionResponseV2
	CheckDelegationRequest     = dtypes.CheckDelegationRequestV2
	CheckDelegationResponse    = dtypes.CheckDelegationResponseV2
	RegisterCandidateRequest   = dtypes.RegisterCandidateRequestV2
	UnregisterCandidateRequest = dtypes.UnregisterCandidateRequestV2
	ListCandidateRequest       = dtypes.ListCandidateRequestV2
	ListCandidateResponse      = dtypes.ListCandidateResponseV2
	ListValidatorsRequest      = dtypes.ListValidatorsRequestV2
	ListValidatorsResponse     = dtypes.ListValidatorsResponseV2
	ElectDelegationRequest     = dtypes.ElectDelegationRequestV2
	Candidate                  = dtypes.CandidateV2
	Delegation                 = dtypes.DelegationV2
	Distribution               = dtypes.DistributionV2
	ValidatorStatistic         = dtypes.ValidatorStatisticV2
	Validator                  = types.Validator
	DposValidator              = dtypes.DposValidator
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

	if params.CoinContractAddress == nil {
		addr, err := ctx.Resolve("coin")
		if err != nil {
			return err
		}
		params.CoinContractAddress = addr.MarshalPB()
	}

	validators := make([]*DposValidator, len(req.Validators), len(req.Validators))
	for i, val := range req.Validators {
		validators[i] = &DposValidator{
			PubKey: val.PubKey,
			// TODO figure out an appropriate dummy value for this
			Power: 12,
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

func (c *DPOS) Delegate(ctx contract.Context, req *DelegateRequest) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	params := state.Params
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)
	coin := &ERC20{
		Context:         ctx,
		ContractAddress: coinAddr,
	}

	delegator := ctx.Message().Sender
	dposContractAddress := ctx.ContractAddress()
	err = coin.TransferFrom(delegator, dposContractAddress, &req.Amount.Value)
	if err != nil {
		return err
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}
	priorDelegation := delegations.Get(*req.ValidatorAddress, *delegator.MarshalPB())

	updatedAmount := loom.BigUInt{big.NewInt(0)}
	if priorDelegation != nil {
		updatedAmount.Add(&priorDelegation.Amount.Value, &req.Amount.Value)
	} else {
		updatedAmount = req.Amount.Value
	}

	delegation := &Delegation{
		Validator: req.ValidatorAddress,
		Delegator: delegator.MarshalPB(),
		Amount:    &types.BigUInt{updatedAmount},
		Height:    uint64(ctx.Block().Height),
	}
	delegations.Set(delegation)

	return saveDelegationList(ctx, delegations)
}

func (c *DPOS) Unbond(ctx contract.Context, req *UnbondRequest) error {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	// TODO abstract this in the three places it appears
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	params := state.Params
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)
	coin := &ERC20{
		Context:         ctx,
		ContractAddress: coinAddr,
	}

	delegator := ctx.Message().Sender

	delegation := delegations.Get(*req.ValidatorAddress, *delegator.MarshalPB())
	if delegation == nil {
		return errors.New(fmt.Sprintf("delegation not found: %s %s", req.ValidatorAddress, delegator.MarshalPB()))
	} else {
		if delegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
			return errors.New("unbond amount exceeds delegation amount")
		} else {
			err = coin.Transfer(delegator, &req.Amount.Value)
			updatedAmount := loom.BigUInt{big.NewInt(0)}
			updatedAmount.Sub(&delegation.Amount.Value, &req.Amount.Value)
			updatedDelegation := &Delegation{
				Delegator: delegator.MarshalPB(),
				Validator: req.ValidatorAddress,
				Amount:    &types.BigUInt{updatedAmount},
				Height:    uint64(ctx.Block().Height),
			}
			delegations.Set(updatedDelegation)
		}
	}

	return saveDelegationList(ctx, delegations)
}

func (c *DPOS) CheckDelegation(ctx contract.StaticContext, req *CheckDelegationRequest) (*CheckDelegationResponse, error) {
	if req.ValidatorAddress == nil {
		return nil, errors.New("CheckDelegation called with req.ValidatorAddress == nil")
	}
	if req.DelegatorAddress == nil {
		return nil, errors.New("CheckDelegation called with req.DelegatorAddress == nil")
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}
	delegation := delegations.Get(*req.ValidatorAddress, *req.DelegatorAddress)
	if delegation == nil {
		return nil, errors.New(fmt.Sprintf("delegation not found: %s %s", req.ValidatorAddress, req.DelegatorAddress))
	} else {
		return &CheckDelegationResponse{delegation}, nil
	}
}

func (c *DPOS) RegisterCandidate(ctx contract.Context, req *RegisterCandidateRequest) error {
	candidateAddress := ctx.Message().Sender
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	checkAddr := loom.LocalAddressFromPublicKey(req.PubKey)
	if candidateAddress.Local.Compare(checkAddr) != 0 {
		return errors.New("public key does not match address")
	}

	newCandidate := &dtypes.CandidateV2{
		PubKey:  req.PubKey,
		Address: candidateAddress.MarshalPB(),
		Fee: req.Fee,
	}
	candidates.Set(newCandidate)
	return saveCandidateList(ctx, candidates)
}

// TODO all slashing must be applied and rewards distributed to delegators
func (c *DPOS) UnregisterCandidate(ctx contract.Context, req *dtypes.UnregisterCandidateRequestV2) error {
	candidateAddress := ctx.Message().Sender
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return errCandidateNotRegistered
	}

	candidates.Delete(candidateAddress)
	return saveCandidateList(ctx, candidates)
}

func (c *DPOS) ListCandidates(ctx contract.StaticContext, req *ListCandidateRequest) (*ListCandidateResponse, error) {
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	return &ListCandidateResponse{
		Candidates: candidates,
	}, nil
}

func (c *DPOS) ElectByDelegation(ctx contract.Context, req *ElectDelegationRequest) error {
	return Elect(ctx)
}

func Elect(ctx contract.Context) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	params := state.Params

	// Check if enough time has elapsed to start new validator election
	if params.ElectionCycleLength < (state.LastElectionTime - ctx.Now().Unix()) {
		return nil
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	// TODO: decide what to do when there are no token delegations.
	// For now, quit the function early and leave the validators as they
	if len(delegations) == 0 {
		return nil
	}

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}
	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return err
	}
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}

	validatorTotals := make(map[string]*loom.BigUInt)
	validatorRewards := make(map[string]*loom.BigUInt)
	for _, validator := range state.Validators {

		// TODO aggregate slashes

		// get candidate record to lookup fee
		candidate := candidates.GetByPubKey(validator.PubKey)

		if candidate != nil {
			validatorKey := loom.UnmarshalAddressPB(candidate.Address).String()
			//get validator statistics
			statistic := statistics.Get(loom.UnmarshalAddressPB(candidate.Address))
			if statistic == nil {
				validatorRewards[validatorKey] = &loom.BigUInt{big.NewInt(0)}
			} else {
				validatorShare := calculateDistributionShare(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, loom.BigUInt{statistic.DistributionTotal.Value.Int})

				// increase validator's delegation
				distributions.IncreaseDistribution(*candidate.Address, validatorShare)

				delegatorShare := validatorShare.Sub(&statistic.DistributionTotal.Value, &validatorShare)
				validatorRewards[validatorKey] = delegatorShare

				// Zeroing out validator's distribution total since it will be transfered
				// to the distributions storage during this `Elect` call.
				// Validators and Delegators both can claim their rewards in the
				// same way when this is true.
				statistic.DistributionTotal = &types.BigUInt{loom.BigUInt{big.NewInt(0)}}
			}
			/*
			if &validator.DelegationTotal.Value != nil {
				validatorTotals[validatorKey] = &validator.DelegationTotal.Value
			}
			*/
		}
	}

	counts := make(map[string]*loom.BigUInt)
	for _, delegation := range delegations {
		validatorKey := loom.UnmarshalAddressPB(delegation.Validator).String()

		if counts[validatorKey] != nil {
			counts[validatorKey].Add(counts[validatorKey], &delegation.Amount.Value)
		} else {
			counts[validatorKey] = &delegation.Amount.Value
		}

		// allocating validator distributions to delegators
		delegationTotal := validatorTotals[validatorKey]
		rewardsTotal := validatorRewards[validatorKey]
		if delegationTotal != nil && rewardsTotal != nil {
			delegatorDistribution := calculateShare(delegation.Amount.Value, *delegationTotal, *rewardsTotal)
			// increase a delegator's distribution
			distributions.IncreaseDistribution(*delegation.Delegator, delegatorDistribution)
		}
	}

	saveDistributionList(ctx, distributions)

	delegationResults := make([]*DelegationResult, 0, len(counts))
	for validator := range counts {
		delegationResults = append(delegationResults, &DelegationResult{
			ValidatorAddress: loom.MustParseAddress(validator),
			DelegationTotal:  *counts[validator],
		})
	}
	sort.Sort(byDelegationTotal(delegationResults))

	// TODO new delegations should probably be integrated at this point

	validatorCount := int(params.ValidatorCount)
	if len(delegationResults) < validatorCount {
		validatorCount = len(delegationResults)
	}

	validators := make([]*DposValidator, 0)
	for _, res := range delegationResults[:validatorCount] {
		candidate := candidates.Get(res.ValidatorAddress)
		if candidate != nil {
			delegationTotal := res.DelegationTotal.Int
			var power big.Int
			// making sure that the validator power can fit into a int64
			power.Div(delegationTotal, big.NewInt(1000000000))
			validatorPower := power.Int64()
			validators = append(validators, &DposValidator{
				PubKey: candidate.PubKey,
				Power:  validatorPower,
				DelegationTotal: &types.BigUInt{res.DelegationTotal},
			})
			statistic := statistics.Get(loom.UnmarshalAddressPB(candidate.Address))
			if statistic == nil {
				statistics = append(statistics, &ValidatorStatistic{
					Address: res.ValidatorAddress.MarshalPB(),
					DistributionTotal: &types.BigUInt{loom.BigUInt{big.NewInt(0)}},
				})
			}
		}
	}

	saveValidatorStatisticList(ctx, statistics)
	state.Validators = sortValidators(validators)
	state.LastElectionTime = ctx.Now().Unix()
	return saveState(ctx, state)
}

func (c *DPOS) ListValidators(ctx contract.StaticContext, req *ListValidatorsRequest) (*ListValidatorsResponse, error) {
	return ValidatorList(ctx)
}

func ValidatorList(ctx contract.StaticContext) (*ListValidatorsResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	return &ListValidatorsResponse{
		Validators: state.Validators,
	}, nil
}

// only called for validators, never delegators
func Reward(ctx contract.Context, validatorAddr loom.Address) error {
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}

	// TODO figure out what a reasonable reward would be
	reward := loom.BigUInt{big.NewInt(100)}
	// update this validator's reward record
	err = statistics.IncreaseValidatorReward(validatorAddr, reward)
	if err != nil {
		return err
	}

	return saveValidatorStatisticList(ctx, statistics)
}

func (c *DPOS) ClaimDistribution(ctx contract.Context, req *ClaimDistributionRequest) (*ClaimDistributionResponse, error) {
	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return nil, err
	}

	delegator := ctx.Message().Sender

	distribution := distributions.Get(*delegator.MarshalPB())
	if distribution == nil {
		return nil, errors.New(fmt.Sprintf("distribution not found: %s", delegator))
	}

	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}
	params := state.Params
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)
	coin := &ERC20{
		Context:         ctx,
		ContractAddress: coinAddr,
	}

	// send distribution to delegator
	err = coin.Transfer(loom.UnmarshalAddressPB(req.WithdrawalAddress), &distribution.Amount.Value)
	if err != nil {
		return nil, err
	}

	claimedAmount := *distribution.Amount
	resp := &ClaimDistributionResponse{Amount: &claimedAmount}

	err = distributions.ResetTotal(*delegator.MarshalPB())
	if err != nil {
		return nil, err
	}

	err = saveDistributionList(ctx, distributions)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// only called for validators, never delegators
func Slash(ctx contract.Context, validatorAddr loom.Address) error {
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&DPOS{})
