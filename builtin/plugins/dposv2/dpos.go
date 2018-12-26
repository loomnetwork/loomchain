package dposv2

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"

	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
)

const (
	registrationRequirement = 1250000
	tokenDecimals           = 18
	yearSeconds             = int64(60 * 60 * 24 * 365)
	BONDING                 = dtypes.DelegationV2_BONDING
	BONDED                  = dtypes.DelegationV2_BONDED
	UNBONDING               = dtypes.DelegationV2_UNBONDING
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
	InitRequest                       = dtypes.DPOSInitRequestV2
	DelegateRequest                   = dtypes.DelegateRequestV2
	WhitelistCandidateRequest         = dtypes.WhitelistCandidateRequestV2
	RemoveWhitelistedCandidateRequest = dtypes.RemoveWhitelistedCandidateRequestV2
	DelegationState                   = dtypes.DelegationV2_DelegationState
	UnbondRequest                     = dtypes.UnbondRequestV2
	ClaimDistributionRequest          = dtypes.ClaimDistributionRequestV2
	ClaimDistributionResponse         = dtypes.ClaimDistributionResponseV2
	CheckDelegationRequest            = dtypes.CheckDelegationRequestV2
	CheckDelegationResponse           = dtypes.CheckDelegationResponseV2
	RegisterCandidateRequest          = dtypes.RegisterCandidateRequestV2
	UnregisterCandidateRequest        = dtypes.UnregisterCandidateRequestV2
	ListCandidateRequest              = dtypes.ListCandidateRequestV2
	ListCandidateResponse             = dtypes.ListCandidateResponseV2
	ListValidatorsRequest             = dtypes.ListValidatorsRequestV2
	ListValidatorsResponse            = dtypes.ListValidatorsResponseV2
	ElectDelegationRequest            = dtypes.ElectDelegationRequestV2
	Candidate                         = dtypes.CandidateV2
	Delegation                        = dtypes.DelegationV2
	Distribution                      = dtypes.DistributionV2
	ValidatorStatistic                = dtypes.ValidatorStatisticV2
	Validator                         = types.Validator
	State                             = dtypes.StateV2
	Params                            = dtypes.ParamsV2

	RequestBatch                = dtypes.RequestBatchV2
	RequestBatchTally           = dtypes.RequestBatchTallyV2
	BatchRequestMeta            = dtypes.BatchRequestMetaV2
	GetRequestBatchTallyRequest = dtypes.GetRequestBatchTallyRequestV2
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

	state := &State{
		Params:     params,
		Validators: req.Validators,
		// we avoid calling ctx.Now() in case the contract is deployed at
		// genesis
		LastElectionTime: 0,
	}

	return saveState(ctx, state)
}

// *********************
// DELEGATION
// *********************

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
		amount = loom.BigZeroPB()
	}

	delegation := &Delegation{
		Validator:    req.ValidatorAddress,
		Delegator:    delegator.MarshalPB(),
		Amount:       amount,
		UpdateAmount: req.Amount,
		Height:       uint64(ctx.Block().Height),
		// delegations are locked up for a minimum of an election period
		// from the time of the latest delegation
		LockTime: uint64(ctx.Now().Unix() + state.Params.ElectionCycleLength),
		State:    BONDING,
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

// **************************
// CANDIDATE REGISTRATION
// **************************

func (c *DPOS) whitelistCandidate(ctx contract.Context, req *WhitelistCandidateRequest) error {
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}

	statistic := statistics.Get(loom.UnmarshalAddressPB(req.CandidateAddress))
	if statistic == nil {
		// Creating a ValidatorStatistic entry for candidate with the appropriate
		// lockup period and amount
		statistics = append(statistics, &ValidatorStatistic{
			Address:           req.CandidateAddress,
			WhitelistAmount:   req.Amount,
			WhitelistLocktime: req.LockTime,
			DistributionTotal: loom.BigZeroPB(),
			DelegationTotal:   loom.BigZeroPB(),
			SlashPercentage:   loom.BigZeroPB(),
		})
	} else {
		// ValidatorStatistic must not yet exist for a particular candidate in order
		// to be whitelisted
		return errors.New("Cannot whitelist an already whitelisted candidate.")
	}

	return saveValidatorStatisticList(ctx, statistics)
}

func (c *DPOS) RemoveWhitelistedCandidate(ctx contract.Context, req *RemoveWhitelistedCandidateRequest) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	sender := ctx.Message().Sender
	if state.Params.OracleAddress != nil && sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return errors.New("Function can only be called with oracle address.")
	}

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}

	statistic := statistics.Get(loom.UnmarshalAddressPB(req.CandidateAddress))

	if statistic == nil {
		return errors.New("Candidate is not whitelisted.")
	} else {
		statistic.WhitelistLocktime = 0
		statistic.WhitelistAmount = loom.BigZeroPB()
	}

	return saveValidatorStatisticList(ctx, statistics)
}

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

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	statistic := statistics.Get(candidateAddress)

	if statistic == nil || common.IsZero(statistic.WhitelistAmount.Value) {
		// A currently unregistered candidate must make a loom token deposit
		// = 'registrationRequirement' in order to run for validator.
		state, err := loadState(ctx)
		if err != nil {
			return err
		}
		coin := loadCoin(ctx, state.Params)

		dposContractAddress := ctx.ContractAddress()
		registrationFee := scientificNotation(registrationRequirement, tokenDecimals)
		err = coin.TransferFrom(candidateAddress, dposContractAddress, registrationFee)
		if err != nil {
			return err
		}

		delegations, err := loadDelegationList(ctx)
		if err != nil {
			return err
		}

		delegation := &Delegation{
			Validator:    candidateAddress.MarshalPB(),
			Delegator:    candidateAddress.MarshalPB(),
			Amount:       loom.BigZeroPB(),
			UpdateAmount: &types.BigUInt{Value: *registrationFee},
			Height:       uint64(ctx.Block().Height),
			// delegations are locked up for a minimum of an election period
			// from the time of the latest delegation
			LockTime: uint64(ctx.Now().Unix() + state.Params.ElectionCycleLength),
			State:    BONDING,
		}
		delegations.Set(delegation)

		err = saveDelegationList(ctx, delegations)
		if err != nil {
			return err
		}
	}

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
// but it should not result in slashing due to downtime.
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

		// reset validator self-delegation
		delegation := delegations.Get(*candidateAddress.MarshalPB(), *candidateAddress.MarshalPB())

		// In case that a whitelisted candidate with no self-delegation calls this
		// function, we must check that delegation is not nil
		if delegation != nil {
			if delegation.LockTime > uint64(ctx.Now().Unix()) {
				return errors.New("Validator's self-delegation currently locked.")
			} else if delegation.State != BONDED {
				return errors.New("Existing delegation not in BONDED state.")
			} else {
				// Once this delegation is unbonded, the total self-delegation
				// amount will be returned to the unregistered validator
				delegation.State = UNBONDING
				delegation.UpdateAmount = &types.BigUInt{Value: delegation.Amount.Value}
				delegations.Set(delegation)
				saveDelegationList(ctx, delegations)
			}
		}
	}

	// Remove canidate from candidates array
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

// ***************************
// ELECTIONS & VALIDATORS
// ***************************

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

	formerValidatorTotals, delegatorRewards := rewardAndSlash(state, candidates, &statistics, &delegations, &distributions)

	newDelegationTotals, err := distributeDelegatorRewards(ctx, *state, formerValidatorTotals, delegatorRewards, &delegations, &distributions, &statistics)
	if err != nil {
		return err
	}
	// save delegation updates that occured in distributeDelegatorRewards
	saveDelegationList(ctx, delegations)
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

			statistic := statistics.Get(loom.UnmarshalAddressPB(candidate.Address))
			if statistic == nil {
				statistics = append(statistics, &ValidatorStatistic{
					Address:           res.ValidatorAddress.MarshalPB(),
					PubKey:            candidate.PubKey,
					DistributionTotal: loom.BigZeroPB(),
					DelegationTotal:   delegationTotal,
					SlashPercentage:   loom.BigZeroPB(),
					WhitelistAmount:   loom.BigZeroPB(),
					WhitelistLocktime: 0,
				})
			} else {
				statistic.DelegationTotal = delegationTotal
				// Needed in case pubkey was not set during whitelisting
				statistic.PubKey = candidate.PubKey
			}
		}
	}

	saveValidatorStatisticList(ctx, statistics)
	state.Validators = validators
	state.LastElectionTime = ctx.Now().Unix()
	return saveState(ctx, state)
}

func (c *DPOS) ListValidators(ctx contract.StaticContext, req *ListValidatorsRequest) (*ListValidatorsResponse, error) {
	validators, err := ValidatorList(ctx)
	if err != nil {
		return nil, err
	}

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return nil, err
	}

	displayStatistics := make([]*ValidatorStatistic, 0)
	for _, validator := range validators {
		address := loom.Address{Local: loom.LocalAddressFromPublicKey(validator.PubKey)}

		// get validator statistics
		stat := statistics.Get(address)
		if stat == nil {
			stat = &ValidatorStatistic{
				PubKey:  validator.PubKey,
				Address: address.MarshalPB(),
			}
		}
		displayStatistics = append(displayStatistics, stat)
	}

	return &ListValidatorsResponse{
		Statistics: displayStatistics,
	}, nil
}

func ValidatorList(ctx contract.StaticContext) ([]*types.Validator, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	return state.Validators, nil
}

// ***************************
// REWARDS & SLASHING
// ***************************

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
	updatedAmount := common.BigZero()
	updatedAmount.Add(&stat.SlashPercentage.Value, &slashPercentage)
	stat.SlashPercentage = &types.BigUInt{Value: *updatedAmount}
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
	delegatorRewards := make(map[string]*loom.BigUInt)
	for _, validator := range state.Validators {
		// get candidate record to lookup fee
		candidate := candidates.GetByPubKey(validator.PubKey)

		if candidate != nil {
			candidateAddress := loom.UnmarshalAddressPB(candidate.Address)
			validatorKey := candidateAddress.String()
			//get validator statistics
			statistic := statistics.Get(candidateAddress)

			if statistic == nil {
				delegatorRewards[validatorKey] = common.BigZero()
				formerValidatorTotals[validatorKey] = *common.BigZero()
			} else {
				// If a validator's SlashPercentage is 0, the validator is
				// rewarded for avoiding faults during the last slashing period
				if common.IsZero(statistic.SlashPercentage.Value) {
					rewardValidator(statistic, state.Params)

					validatorShare := calculateDistributionShare(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, statistic.DistributionTotal.Value)

					// increase validator's delegation
					distributions.IncreaseDistribution(*candidate.Address, validatorShare)

					// delegatorsShare is the amount to all delegators in proportion
					// to the amount that they've delegatored
					delegatorsShare := common.BigZero()
					delegatorsShare.Sub(&statistic.DistributionTotal.Value, &validatorShare)
					delegatorRewards[validatorKey] = delegatorsShare

					// If a validator has some non-zero WhitelistAmount,
					// calculate the validator's reward based on whitelist amount & locktime
					if !common.IsZero(statistic.WhitelistAmount.Value) {
						whitelistDistribution := calculateShare(statistic.WhitelistAmount.Value, statistic.DelegationTotal.Value, *delegatorsShare)
						// increase a delegator's distribution
						distributions.IncreaseDistribution(*candidate.Address, whitelistDistribution)
					}
				} else {
					slashValidatorDelegations(delegations, statistic, candidateAddress)
				}

				// Zeroing out validator's distribution total since it will be transfered
				// to the distributions storage during this `Elect` call.
				// Validators and Delegators both can claim their rewards in the
				// same way when this is true.
				statistic.DistributionTotal = &types.BigUInt{Value: *common.BigZero()}
				formerValidatorTotals[validatorKey] = statistic.DelegationTotal.Value
			}
		}
	}
	return formerValidatorTotals, delegatorRewards
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
	updatedAmount := common.BigZero()
	updatedAmount.Add(&statistic.DistributionTotal.Value, &reward)
	statistic.DistributionTotal = &types.BigUInt{Value: *updatedAmount}
	return
}

func slashValidatorDelegations(delegations *DelegationList, statistic *ValidatorStatistic, validatorAddress loom.Address) {
	// these delegation totals will be added back up again when we calculate new delegation totals below
	for _, delegation := range *delegations {
		// check the it's a delegation that belongs to the validator
		if delegation.Validator.Local.Compare(validatorAddress.Local) == 0 {
			toSlash := calculateDistributionShare(statistic.SlashPercentage.Value, delegation.Amount.Value)
			updatedAmount := common.BigZero()
			updatedAmount.Sub(&delegation.Amount.Value, &toSlash)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
		}
	}

	// Slash a whitelisted candidate's whitelist amount. This doesn't affect how
	// much the validator gets back from token timelock, but will decrease the
	// validator's delegation total & thus his ability to earn rewards
	if !common.IsZero(statistic.WhitelistAmount.Value) {
		toSlash := calculateDistributionShare(statistic.SlashPercentage.Value, statistic.WhitelistAmount.Value)
		updatedAmount := common.BigZero()
		updatedAmount.Sub(&statistic.WhitelistAmount.Value, &toSlash)
		statistic.WhitelistAmount = &types.BigUInt{Value: *updatedAmount}
	}

	// reset slash total
	statistic.SlashPercentage = loom.BigZeroPB()
}

// This function has three goals 1) distribute a validator's rewards to each of
// the delegators, 2) finalize the bonding process for any delegations recieved
// during the last election period (delegate & unbond calls) and 3) calculate
// the new delegation totals.
func distributeDelegatorRewards(ctx contract.Context, state State, formerValidatorTotals map[string]loom.BigUInt, delegatorRewards map[string]*loom.BigUInt, delegations *DelegationList, distributions *DistributionList, statistics *ValidatorStatisticList) (map[string]*loom.BigUInt, error) {
	newDelegationTotals := make(map[string]*loom.BigUInt)

	// initialize delegation totals with whitelist amounts
	for _, statistic := range *statistics {
		if statistic.WhitelistAmount != nil && !common.IsZero(statistic.WhitelistAmount.Value) {
			validatorKey := loom.UnmarshalAddressPB(statistic.Address).String()
			newDelegationTotals[validatorKey] = &statistic.WhitelistAmount.Value
		}
	}

	for _, delegation := range *delegations {
		validatorKey := loom.UnmarshalAddressPB(delegation.Validator).String()

		// allocating validator distributions to delegators
		// based on former validator delegation totals
		delegationTotal := formerValidatorTotals[validatorKey]
		rewardsTotal := delegatorRewards[validatorKey]
		if rewardsTotal != nil {
			delegatorDistribution := calculateShare(delegation.Amount.Value, delegationTotal, *rewardsTotal)
			// increase a delegator's distribution
			distributions.IncreaseDistribution(*delegation.Delegator, delegatorDistribution)
		}

		updatedAmount := common.BigZero()
		if delegation.State == BONDING {
			updatedAmount.Add(&delegation.Amount.Value, &delegation.UpdateAmount.Value)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
		} else if delegation.State == UNBONDING {
			updatedAmount.Sub(&delegation.Amount.Value, &delegation.UpdateAmount.Value)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
			coin := loadCoin(ctx, state.Params)
			err := coin.Transfer(loom.UnmarshalAddressPB(delegation.Delegator), &delegation.UpdateAmount.Value)
			if err != nil {
				return nil, err
			}
		}
		// After a delegation update, zero out UpdateAmount
		delegation.UpdateAmount = loom.BigZeroPB()
		delegation.State = BONDED

		newTotal := common.BigZero()
		newTotal.Add(newTotal, &delegation.Amount.Value)
		if newDelegationTotals[validatorKey] != nil {
			newTotal.Add(newTotal, newDelegationTotals[validatorKey])
		}
		newDelegationTotals[validatorKey] = newTotal
	}

	return newDelegationTotals, nil
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

	claimedAmount := common.BigZero()
	claimedAmount.Add(&distribution.Amount.Value, claimedAmount)
	resp := &ClaimDistributionResponse{Amount: &types.BigUInt{Value: *claimedAmount}}

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

//
// Oracle methods
//

func (c *DPOS) GetRequestBatchTally(ctx contract.StaticContext, req *GetRequestBatchTallyRequest) (*RequestBatchTally, error) {
	return loadRequestBatchTally(ctx)
}

func isRequestAlreadySeen(meta *BatchRequestMeta, currentTally *RequestBatchTally) bool {
	if meta.BlockNumber != currentTally.LastSeenBlockNumber {
		return meta.BlockNumber <= currentTally.LastSeenBlockNumber
	}

	if meta.TxIndex != currentTally.LastSeenTxIndex {
		return meta.TxIndex <= currentTally.LastSeenTxIndex
	}

	if meta.LogIndex != currentTally.LastSeenLogIndex {
		return meta.LogIndex <= currentTally.LastSeenLogIndex
	}

	return true
}

func (c *DPOS) ProcessRequestBatch(ctx contract.Context, req *RequestBatch) error {
	// ensure that function is only executed when called by oracle
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	sender := ctx.Message().Sender
	if state.Params.OracleAddress != nil && sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return errors.New("[ProcessRequestBatch] only oracle is authorized to call ProcessRequestBatch")
	}

	if req.Batch == nil || len(req.Batch) == 0 {
		return fmt.Errorf("[ProcessRequestBatch] invalid Request, no batch request found")
	}

	tally, err := loadRequestBatchTally(ctx)
	if err != nil {
		return err
	}

	lastRequest := req.Batch[len(req.Batch)-1]
	if isRequestAlreadySeen(lastRequest.Meta, tally) {
		return fmt.Errorf("[ProcessRequestBatch] invalid Request, all events has been already seen")
	}

loop:
	for _, request := range req.Batch {
		switch payload := request.Payload.(type) {
		case *dtypes.BatchRequestV2_WhitelistCandidate:
			if isRequestAlreadySeen(request.Meta, tally) {
				break
			}

			if err = c.whitelistCandidate(ctx, payload.WhitelistCandidate); err != nil {
				break loop
			}

			tally.LastSeenBlockNumber = request.Meta.BlockNumber
			tally.LastSeenTxIndex = request.Meta.TxIndex
			tally.LastSeenLogIndex = request.Meta.LogIndex
		default:
			err = fmt.Errorf("unsupported type of request in request batch")
		}
	}

	if err != nil {
		return fmt.Errorf("[ProcessRequestBatch] unable to consume one or more request, error: %v", err)
	}

	if err = saveRequestBatchTally(ctx, tally); err != nil {
		return fmt.Errorf("[ProcessRequestBatch] unable to save request tally, error: %v", err)
	}

	return nil
}
