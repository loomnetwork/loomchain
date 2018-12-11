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

const (
	yearSeconds               = int64(60 * 60 * 24 * 365)
	BONDING                   = dtypes.DelegationV2_BONDING
	BONDED                    = dtypes.DelegationV2_BONDED
	UNBONDING                 = dtypes.DelegationV2_UNBONDING
)

var (
	secondsInYear             = loom.BigUInt{big.NewInt(yearSeconds)}
	basisPoints               = loom.BigUInt{big.NewInt(10000)}
	blockRewardPercentage     = loom.BigUInt{big.NewInt(700)}
	doubleSignSlashPercentage = loom.BigUInt{big.NewInt(500)}
	inactivitySlashPercentage = loom.BigUInt{big.NewInt(100)}
	powerCorrection           = big.NewInt(1000000000)
	errCandidateNotRegistered = errors.New("candidate is not registered")
	errValidatorNotFound      = errors.New("validator not found")
	errDistributionNotFound   = errors.New("distribution not found")
)

type (
	InitRequest                = dtypes.DPOSInitRequestV2
	DelegateRequest            = dtypes.DelegateRequestV2
	DelegationOverrideRequest  = dtypes.DelegationOverrideRequestV2
	DelegationState            = dtypes.DelegationV2_DelegationState
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

	sortedValidators := sortValidators(req.Validators)

	state := &State{
		Params:           params,
		Validators:       sortedValidators,
		// we avoid calling ctx.Now() in case the contract is deployed at
		// genesis
		LastElectionTime: 0,
	}

	return saveState(ctx, state)
}

func (c *DPOS) Delegate(ctx contract.Context, req *DelegateRequest) error {
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}
	cand := candidates.Get(loom.UnmarshalAddressPB(req.ValidatorAddress))
	// Delegations can only be made to existing candidates
	if cand == nil {
		return errors.New("Candidate record does not exist.")
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	coin := loadCoin(ctx, state.Params)

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

	var amount *types.BigUInt
	if priorDelegation != nil {
		if priorDelegation.State != BONDED {
			return errors.New("Existing delegation not in BONDED state.")
		}
		amount = priorDelegation.Amount
	} else {
		amount = &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}}
	}

	delegation := &Delegation{
		Validator:    req.ValidatorAddress,
		Delegator:    delegator.MarshalPB(),
		Amount:       amount,
		UpdateAmount: req.Amount,
		Height:       uint64(ctx.Block().Height),
		// delegations are locked up for a minimum of an election period
		// from the time of the latest delegation
		LockTime:     uint64(ctx.Now().Unix() + state.Params.ElectionCycleLength),
		State:        BONDING,
	}
	delegations.Set(delegation)

	return saveDelegationList(ctx, delegations)
}

func (c *DPOS) DelegationOverride(ctx contract.Context, req *DelegationOverrideRequest) error {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}
	priorDelegation := delegations.Get(*req.ValidatorAddress, *req.ValidatorAddress)

	updateAmount := loom.BigUInt{big.NewInt(0)}
	updateAmount.Sub(&priorDelegation.Amount.Value, &req.Amount.Value)
	delegation := &Delegation{
		Validator:    req.ValidatorAddress,
		Delegator:    req.DelegatorAddress,
		Amount:       priorDelegation.Amount,
		UpdateAmount: &types.BigUInt{Value: updateAmount},
		Height:       uint64(ctx.Block().Height),
		// delegations are locked up for a minimum of an election period
		// from the time of the latest delegation
		LockTime:     uint64(req.LockTime),
		State:        BONDING,
	}
	delegations.Set(delegation)

	return saveDelegationList(ctx, delegations)
}

func (c *DPOS) Unbond(ctx contract.Context, req *UnbondRequest) error {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	delegator := ctx.Message().Sender

	delegation := delegations.Get(*req.ValidatorAddress, *delegator.MarshalPB())
	if delegation == nil {
		return errors.New(fmt.Sprintf("delegation not found: %s %s", req.ValidatorAddress, delegator.MarshalPB()))
	} else {
		if delegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
			return errors.New("Unbond amount exceeds delegation amount.")
		} else if delegation.LockTime > uint64(ctx.Now().Unix()) {
			return errors.New("Delegation currently locked.")
		} else if delegation.State != BONDED {
			return errors.New("Existing delegation not in BONDED state.")
		} else {
			delegation.State = UNBONDING
			delegation.UpdateAmount = req.Amount
			delegations.Set(delegation)
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
		return &CheckDelegationResponse{Delegation: delegation}, nil
	}
}

// TODO create UpdateCandidateRecord function

func (c *DPOS) RegisterCandidate(ctx contract.Context, req *RegisterCandidateRequest) error {
	candidateAddress := ctx.Message().Sender
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	checkAddr := loom.LocalAddressFromPublicKey(req.PubKey)
	if candidateAddress.Local.Compare(checkAddr) != 0 {
		return errors.New("Public key does not match address.")
	}

	// if candidate record already exists, exit function; candidate record
	// updates are done via the UpdateCandidateRecord function
	cand := candidates.Get(candidateAddress)
	if cand != nil {
		return errors.New("Candidate record already exists.")
	}

	// TODO a currently unregistered candidate which must make a ~1.25M loom
	// token deposit in order to run for validator.

	newCandidate := &dtypes.CandidateV2{
		PubKey:      req.PubKey,
		Address:     candidateAddress.MarshalPB(),
		Fee:         req.Fee,
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
	}
	candidates.Set(newCandidate)
	return saveCandidateList(ctx, candidates)
}

// When UnregisterCandidate is called, all slashing must be applied to
// delegators. Delegators can be unbonded AFTER SOME WITHDRAWAL DELAY.
// Leaving the validator set mid-election period results in a loss of rewards
// but it should not result in slashing due to downtime. TODO this must be tested
func (c *DPOS) UnregisterCandidate(ctx contract.Context, req *dtypes.UnregisterCandidateRequestV2) error {
	candidateAddress := ctx.Message().Sender
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return errCandidateNotRegistered
	} else {
		delegations, err := loadDelegationList(ctx)
		if err != nil {
			return err
		}

		statistics, err := loadValidatorStatisticList(ctx)
		if err != nil {
			return err
		}
		statistic := statistics.Get(candidateAddress)

		slashValidatorDelegations(&delegations, statistic, candidateAddress)
	}

	candidates.Delete(candidateAddress)
	// TODO return ~1.25M loom token deposit required of all candidates

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

// electing and settling rewards settlement
func Elect(ctx contract.Context) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// Check if enough time has elapsed to start new validator election
	if state.Params.ElectionCycleLength > (ctx.Now().Unix() - state.LastElectionTime) {
		return nil
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	// When there are no token delegations, quit the function early
	// and leave the validators as they are
	if len(delegations) == 0 {
		return nil
	}

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}
	// If there are no candidates, do not run election
	if len(candidates) == 0 {
		return nil
	}


	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return err
	}
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}

	formerValidatorTotals, validatorRewards := rewardAndSlash(state, candidates, &statistics, &delegations, &distributions)

	newDelegationTotals, err := distributeDelegatorRewards(ctx, *state, formerValidatorTotals, validatorRewards, &delegations, &distributions)
	if err != nil {
		return err
	}

	saveDistributionList(ctx, distributions)

	delegationResults := make([]*DelegationResult, 0, len(newDelegationTotals))
	for validator := range newDelegationTotals {
		delegationResults = append(delegationResults, &DelegationResult{
			ValidatorAddress: loom.MustParseAddress(validator),
			DelegationTotal:  *newDelegationTotals[validator],
		})
	}
	sort.Sort(byDelegationTotal(delegationResults))

	validatorCount := int(state.Params.ValidatorCount)
	if len(delegationResults) < validatorCount {
		validatorCount = len(delegationResults)
	}

	validators := make([]*Validator, 0)
	for _, res := range delegationResults[:validatorCount] {
		candidate := candidates.Get(res.ValidatorAddress)
		if candidate != nil {
			var power big.Int
			// making sure that the validator power can fit into a int64
			power.Div(res.DelegationTotal.Int, powerCorrection)
			validatorPower := power.Int64()
			delegationTotal := &types.BigUInt{Value: res.DelegationTotal}
			validators = append(validators, &Validator{
				PubKey: candidate.PubKey,
				Power:  validatorPower,
			})
			// TODO abstract into function
			statistic := statistics.Get(loom.UnmarshalAddressPB(candidate.Address))
			if statistic == nil {
				statistics = append(statistics, &ValidatorStatistic{
					Address:           res.ValidatorAddress.MarshalPB(),
					PubKey:            candidate.PubKey,
					DistributionTotal: &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}},
					DelegationTotal:   delegationTotal,
					SlashPercentage:   &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}},
				})
			} else {
				statistic.DelegationTotal = delegationTotal
			}
		}
	}

	saveValidatorStatisticList(ctx, statistics)
	state.Validators = sortValidators(validators)
	saveDelegationList(ctx, delegations)
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
	coin := loadCoin(ctx, state.Params)

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
func SlashInactivity(ctx contract.Context, validatorAddr []byte) error {
	return slash(ctx, validatorAddr, inactivitySlashPercentage)
}

func SlashDoubleSign(ctx contract.Context, validatorAddr []byte) error {
	return slash(ctx, validatorAddr, doubleSignSlashPercentage)
}

func slash(ctx contract.Context, validatorAddr []byte, slashPercentage loom.BigUInt) error {
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	stat := statistics.GetV2(validatorAddr)
	if stat == nil {
		return errors.New("Cannot slash default validator.")
	}
	updatedAmount := loom.BigUInt{big.NewInt(0)}
	updatedAmount.Add(&stat.SlashPercentage.Value, &slashPercentage)
	stat.SlashPercentage = &types.BigUInt{Value: updatedAmount}
	return saveValidatorStatisticList(ctx, statistics)
}

var Contract plugin.Contract = contract.MakePluginContract(&DPOS{})

// UTILITIES

func loadCoin(ctx contract.Context, params *Params) *ERC20 {
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)
	return &ERC20{
		Context:         ctx,
		ContractAddress: coinAddr,
	}
}

// rewards & slashes are calculated along with former delegation totals
// rewards are distributed to validators based on fee
// rewards distribution amounts are prepared for delegators
func rewardAndSlash(state *State, candidates CandidateList, statistics *ValidatorStatisticList, delegations *DelegationList, distributions *DistributionList) (map[string]loom.BigUInt, map[string]*loom.BigUInt) {
	formerValidatorTotals := make(map[string]loom.BigUInt)
	validatorRewards := make(map[string]*loom.BigUInt)
	for _, validator := range state.Validators {
		// get candidate record to lookup fee
		candidate := candidates.GetByPubKey(validator.PubKey)

		if candidate != nil {
			candidateAddress := loom.UnmarshalAddressPB(candidate.Address)
			validatorKey := candidateAddress.String()
			//get validator statistics
			statistic := statistics.Get(candidateAddress)

			if statistic == nil {
				validatorRewards[validatorKey] = &loom.BigUInt{big.NewInt(0)}
				formerValidatorTotals[validatorKey] = loom.BigUInt{big.NewInt(0)}
			} else {
				if statistic.SlashPercentage.Value.Cmp(&loom.BigUInt{big.NewInt(0)}) == 0 {
					rewardValidator(statistic, state.Params)
				} else {
					slashValidatorDelegations(delegations, statistic, candidateAddress)
				}

				validatorShare := calculateDistributionShare(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, statistic.DistributionTotal.Value)

				// increase validator's delegation
				distributions.IncreaseDistribution(*candidate.Address, validatorShare)

				// delegatorsShare is the amount to all delegators in proportion
				// to the amount that they've delegatored
				delegatorsShare := validatorShare.Sub(&statistic.DistributionTotal.Value, &validatorShare)
				validatorRewards[validatorKey] = delegatorsShare

				// Zeroing out validator's distribution total since it will be transfered
				// to the distributions storage during this `Elect` call.
				// Validators and Delegators both can claim their rewards in the
				// same way when this is true.
				statistic.DistributionTotal = &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}}
				formerValidatorTotals[validatorKey] = statistic.DelegationTotal.Value
			}
		}
	}
	return formerValidatorTotals, validatorRewards
}

func rewardValidator(statistic *ValidatorStatistic, params *Params) {
	// if there is no slashing to be applied, reward validator
	cycleSeconds := params.ElectionCycleLength
	reward := calculateDistributionShare(blockRewardPercentage, statistic.DelegationTotal.Value)
	// when election cycle = 0, estimate block time at 2 sec
	if cycleSeconds == 0 {
		cycleSeconds = 2
	}
	reward.Mul(&reward, &loom.BigUInt{big.NewInt(cycleSeconds)})
	reward.Div(&reward, &secondsInYear)
	updatedAmount := loom.BigUInt{big.NewInt(0)}
	updatedAmount.Add(&statistic.DistributionTotal.Value, &reward)
	statistic.DistributionTotal = &types.BigUInt{Value: updatedAmount}
	return
}

func slashValidatorDelegations(delegations *DelegationList, statistic *ValidatorStatistic, validatorAddress loom.Address) {
	// these delegation totals will be added back up again when we calculate new delegation totals below
	for _, delegation := range *delegations {
		// check the it's a delegation that belongs to the validator
		if delegation.Validator.Local.Compare(validatorAddress.Local) == 0 {
			toSlash := calculateDistributionShare(statistic.SlashPercentage.Value, delegation.Amount.Value)
			updatedAmount := loom.BigUInt{big.NewInt(0)}
			updatedAmount.Sub(&delegation.Amount.Value, &toSlash)
			delegation.Amount = &types.BigUInt{Value: updatedAmount}
		}
	}
	// reset slash total
	statistic.SlashPercentage = &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}}
}

// This function has three goals 1) distribute a validator's rewards to each of
// the delegators, 2) finalize the bonding process for any delegations recieved
// during the last election period (delegate & unbond calls) and 3) calculate
// the new delegation totals.
func distributeDelegatorRewards(ctx contract.Context, state State, formerValidatorTotals map[string]loom.BigUInt, validatorRewards map[string]*loom.BigUInt, delegations *DelegationList, distributions *DistributionList) (map[string]*loom.BigUInt, error) {
	newDelegationTotals := make(map[string]*loom.BigUInt)
	for _, delegation := range *delegations {
		validatorKey := loom.UnmarshalAddressPB(delegation.Validator).String()

		// allocating validator distributions to delegators
		// based on former validator delegation totals
		delegationTotal := formerValidatorTotals[validatorKey]
		rewardsTotal := validatorRewards[validatorKey]
		if rewardsTotal != nil {
			delegatorDistribution := calculateShare(delegation.Amount.Value, delegationTotal, *rewardsTotal)
			// increase a delegator's distribution
			distributions.IncreaseDistribution(*delegation.Delegator, delegatorDistribution)
		}

		if delegation.State == BONDING {
			updatedAmount := loom.BigUInt{big.NewInt(0)}
			updatedAmount.Add(&delegation.Amount.Value, &delegation.UpdateAmount.Value)
			delegation.Amount = &types.BigUInt{Value: updatedAmount}
			delegation.State = BONDED
		} else if delegation.State == UNBONDING {
			updatedAmount := loom.BigUInt{big.NewInt(0)}
			updatedAmount.Sub(&delegation.Amount.Value, &delegation.UpdateAmount.Value)
			delegation.Amount = &types.BigUInt{Value: updatedAmount}
			delegation.State = BONDED
			delegation.Height = uint64(ctx.Block().Height)

			coin := loadCoin(ctx, state.Params)
			err := coin.Transfer(loom.UnmarshalAddressPB(delegation.Delegator), &delegation.UpdateAmount.Value)
			if err != nil {
				return nil, err
			}
		}

		if newDelegationTotals[validatorKey] != nil {
			newDelegationTotals[validatorKey].Add(newDelegationTotals[validatorKey], &delegation.Amount.Value)
		} else {
			newDelegationTotals[validatorKey] = &delegation.Amount.Value
		}
	}
	return newDelegationTotals, nil
}
