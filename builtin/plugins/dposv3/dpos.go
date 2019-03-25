package dposv3

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
)

const (
	defaultRegistrationRequirement = 1250000
	defaultMaxYearlyReward         = 60000000
	tokenDecimals                  = 18
	yearSeconds                    = int64(60 * 60 * 24 * 365)
	BONDING                        = dtypes.Delegation_BONDING
	BONDED                         = dtypes.Delegation_BONDED
	UNBONDING                      = dtypes.Delegation_UNBONDING
	REDELEGATING                   = dtypes.Delegation_REDELEGATING
	REGISTERED                     = dtypes.Candidate_REGISTERED
	UNREGISTERING                  = dtypes.Candidate_UNREGISTERING
	ABOUT_TO_CHANGE_FEE            = dtypes.Candidate_ABOUT_TO_CHANGE_FEE
	CHANGING_FEE                   = dtypes.Candidate_CHANGING_FEE
	TIER_ZERO                      = dtypes.Delegation_TIER_ZERO
	TIER_ONE                       = dtypes.Delegation_TIER_ONE
	TIER_TWO                       = dtypes.Delegation_TIER_TWO
	TIER_THREE                     = dtypes.Delegation_TIER_THREE

	ElectionEventTopic              = "dpos:election"
	SlashEventTopic                 = "dpos:slash"
	CandidateRegistersEventTopic    = "dpos:candidateregisters"
	CandidateUnregistersEventTopic  = "dpos:candidateunregisters"
	CandidateFeeChangeEventTopic    = "dpos:candidatefeechange"
	UpdateCandidateInfoEventTopic   = "dpos:updatecandidateinfo"
	DelegatorDelegatesEventTopic    = "dpos:delegatordelegates"
	DelegatorRedelegatesEventTopic  = "dpos:delegatorredelegates"
	DelegatorConsolidatesEventTopic = "dpos:delegatorconsolidates"
	DelegatorUnbondsEventTopic      = "dpos:delegatorunbonds"
)

var (
	secondsInYear                 = loom.BigUInt{big.NewInt(yearSeconds)}
	basisPoints                   = loom.BigUInt{big.NewInt(10000)}
	blockRewardPercentage         = loom.BigUInt{big.NewInt(500)}
	doubleSignSlashPercentage     = loom.BigUInt{big.NewInt(500)}
	inactivitySlashPercentage     = loom.BigUInt{big.NewInt(100)}
	limboValidatorAddress         = loom.MustParseAddress("limbo:0x0000000000000000000000000000000000000000")
	powerCorrection               = big.NewInt(1000000000000)
	errCandidateNotFound          = errors.New("Candidate record not found.")
	errCandidateAlreadyRegistered = errors.New("candidate already registered")
	errValidatorNotFound          = errors.New("Validator record not found.")
	errDistributionNotFound       = errors.New("Distribution record not found.")
	errOnlyOracle                 = errors.New("Function can only be called with oracle address.")
)

type (
	InitRequest                       = dtypes.DPOSInitRequest
	DelegateRequest                   = dtypes.DelegateRequest
	RedelegateRequest                 = dtypes.RedelegateRequest
	WhitelistCandidateRequest         = dtypes.WhitelistCandidateRequest
	RemoveWhitelistedCandidateRequest = dtypes.RemoveWhitelistedCandidateRequest
	ChangeWhitelistAmountRequest      = dtypes.ChangeWhitelistAmountRequest
	DelegationState                   = dtypes.Delegation_DelegationState
	LocktimeTier                      = dtypes.Delegation_LocktimeTier
	UnbondRequest                     = dtypes.UnbondRequest
	ConsolidateDelegationsRequest     = dtypes.ConsolidateDelegationsRequest
	CheckAllDelegationsRequest        = dtypes.CheckAllDelegationsRequest
	CheckAllDelegationsResponse       = dtypes.CheckAllDelegationsResponse
	CheckDelegationRequest            = dtypes.CheckDelegationRequest
	CheckDelegationResponse           = dtypes.CheckDelegationResponse
	TotalDelegationRequest            = dtypes.TotalDelegationRequest
	TotalDelegationResponse           = dtypes.TotalDelegationResponse
	CheckRewardsRequest               = dtypes.CheckRewardsRequest
	CheckRewardsResponse              = dtypes.CheckRewardsResponse
	CheckRewardDelegationRequest      = dtypes.CheckRewardDelegationRequest
	CheckRewardDelegationResponse     = dtypes.CheckRewardDelegationResponse
	TimeUntilElectionRequest          = dtypes.TimeUntilElectionRequest
	TimeUntilElectionResponse         = dtypes.TimeUntilElectionResponse
	RegisterCandidateRequest          = dtypes.RegisterCandidateRequest
	ChangeCandidateFeeRequest         = dtypes.ChangeCandidateFeeRequest
	UpdateCandidateInfoRequest        = dtypes.UpdateCandidateInfoRequest
	UnregisterCandidateRequest        = dtypes.UnregisterCandidateRequest
	ListCandidateRequest              = dtypes.ListCandidateRequest
	ListCandidateResponse             = dtypes.ListCandidateResponse
	ListValidatorsRequest             = dtypes.ListValidatorsRequest
	ListValidatorsResponse            = dtypes.ListValidatorsResponse
	ListDelegationsRequest            = dtypes.ListDelegationsRequest
	ListDelegationsResponse           = dtypes.ListDelegationsResponse
	ListAllDelegationsRequest         = dtypes.ListAllDelegationsRequest
	ListAllDelegationsResponse        = dtypes.ListAllDelegationsResponse
	SetElectionCycleRequest           = dtypes.SetElectionCycleRequest
	SetMaxYearlyRewardRequest         = dtypes.SetMaxYearlyRewardRequest
	SetRegistrationRequirementRequest = dtypes.SetRegistrationRequirementRequest
	SetValidatorCountRequest          = dtypes.SetValidatorCountRequest
	SetOracleAddressRequest           = dtypes.SetOracleAddressRequest
	SetSlashingPercentagesRequest     = dtypes.SetSlashingPercentagesRequest
	Candidate                         = dtypes.Candidate
	CandidateStatistic                = dtypes.CandidateStatistic
	Delegation                        = dtypes.Delegation
	DelegationIndex                   = dtypes.DelegationIndex
	ValidatorStatistic                = dtypes.ValidatorStatistic
	Validator                         = types.Validator
	State                             = dtypes.State
	Params                            = dtypes.Params
	GetStateRequest                   = dtypes.GetStateRequest
	GetStateResponse                  = dtypes.GetStateResponse
	InitializationState               = dtypes.InitializationState

	DposElectionEvent              = dtypes.DposElectionEvent
	DposSlashEvent                 = dtypes.DposSlashEvent
	DposCandidateRegistersEvent    = dtypes.DposCandidateRegistersEvent
	DposCandidateUnregistersEvent  = dtypes.DposCandidateUnregistersEvent
	DposCandidateFeeChangeEvent    = dtypes.DposCandidateFeeChangeEvent
	DposUpdateCandidateInfoEvent   = dtypes.DposUpdateCandidateInfoEvent
	DposDelegatorDelegatesEvent    = dtypes.DposDelegatorDelegatesEvent
	DposDelegatorRedelegatesEvent  = dtypes.DposDelegatorRedelegatesEvent
	DposDelegatorConsolidatesEvent = dtypes.DposDelegatorConsolidatesEvent
	DposDelegatorUnbondsEvent      = dtypes.DposDelegatorUnbondsEvent

	RequestBatch                = dtypes.RequestBatch
	RequestBatchTally           = dtypes.RequestBatchTally
	BatchRequest                = dtypes.BatchRequest
	BatchRequestMeta            = dtypes.BatchRequestMeta
	GetRequestBatchTallyRequest = dtypes.GetRequestBatchTallyRequest
)

type DPOS struct {
}

func (c *DPOS) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "dposV3",
		Version: "3.0.0",
	}, nil
}

func (c *DPOS) Init(ctx contract.Context, req *InitRequest) error {
	ctx.Logger().Info("DPOS Init", "Params", req)
	params := req.Params

	if params.CoinContractAddress == nil {
		addr, err := ctx.Resolve("coin")
		if err != nil {
			return err
		}
		params.CoinContractAddress = addr.MarshalPB()
	}
	if params.CrashSlashingPercentage == nil {
		params.CrashSlashingPercentage = &types.BigUInt{Value: inactivitySlashPercentage}
	}
	if params.ByzantineSlashingPercentage == nil {
		params.ByzantineSlashingPercentage = &types.BigUInt{Value: doubleSignSlashPercentage}
	}
	if params.RegistrationRequirement == nil {
		params.RegistrationRequirement = &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}
	}
	if params.MaxYearlyReward == nil {
		params.MaxYearlyReward = &types.BigUInt{Value: *scientificNotation(defaultMaxYearlyReward, tokenDecimals)}
	}

	state := &State{
		Params:     params,
		Validators: req.Validators,
		// we avoid calling ctx.Now() in case the contract is deployed at
		// genesis
		LastElectionTime:          0,
		TotalValidatorDelegations: loom.BigZeroPB(),
		TotalRewardDistribution:   loom.BigZeroPB(),
	}

	return saveState(ctx, state)
}

// *********************
// DELEGATION
// *********************

func (c *DPOS) Delegate(ctx contract.Context, req *DelegateRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS Delegate", "delegator", delegator, "request", req)

	cand := GetCandidate(ctx, loom.UnmarshalAddressPB(req.ValidatorAddress))
	// Delegations can only be made to existing candidates
	if cand == nil {
		return logDposError(ctx, errCandidateNotFound, req.String())
	}
	if req.Amount == nil || !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Must Delegate a positive number of tokens."), req.String())
	}

	coin, err := loadCoin(ctx)
	if err != nil {
		return err
	}

	dposContractAddress := ctx.ContractAddress()
	err = coin.TransferFrom(delegator, dposContractAddress, &req.Amount.Value)
	if err != nil {
		return err
	}

	// Get next delegation index for this validator / delegator pair
	index, err := GetNextDelegationIndex(ctx, *req.ValidatorAddress, *delegator.MarshalPB())
	if err != nil {
		return err
	}

	tierNumber := req.GetLocktimeTier()
	if tierNumber > 3 {
		return logDposError(ctx, errors.New("Invalid delegation tier"), req.String())
	}

	locktimeTier := TierMap[tierNumber]

	tierTime := TierLocktimeMap[locktimeTier]
	now := uint64(ctx.Now().Unix())
	lockTime := now + tierTime

	if lockTime < now {
		return logDposError(ctx, errors.New("Overflow in set locktime!"), req.String())
	}

	delegation := &Delegation{
		Validator:    req.ValidatorAddress,
		Delegator:    delegator.MarshalPB(),
		Amount:       loom.BigZeroPB(),
		UpdateAmount: req.Amount,
		// delegations are locked up for a minimum of an election period
		// from the time of the latest delegation
		LocktimeTier: locktimeTier,
		LockTime:     lockTime,
		State:        BONDING,
		Index:        index,
	}
	if err := SetDelegation(ctx, delegation); err != nil {
		return err
	}

	return c.emitDelegatorDelegatesEvent(ctx, delegator.MarshalPB(), req.Amount, req.Referrer)
}

func (c *DPOS) Redelegate(ctx contract.Context, req *RedelegateRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS Redelegate", "delegator", delegator, "request", req)

	if req.FormerValidatorAddress.Local.Compare(req.ValidatorAddress.Local) == 0 {
		return logDposError(ctx, errors.New("Redelegating self-delegations is not permitted."), req.String())
	}
	if req.Amount != nil && !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Must Redelegate a positive number of tokens."), req.String())
	}

	// Unless redelegation is to the limbo validator check that the new
	// validator address corresponds to one of the registered candidates
	if req.ValidatorAddress.Local.Compare(limboValidatorAddress.Local) != 0 {
		candidate := GetCandidate(ctx, loom.UnmarshalAddressPB(req.ValidatorAddress))
		// Delegations can only be made to existing candidates
		if candidate == nil {
			return logDposError(ctx, errCandidateNotFound, req.String())
		}
	}

	priorDelegation, err := GetDelegation(ctx, req.Index, *req.FormerValidatorAddress, *delegator.MarshalPB())
	if err == contract.ErrNotFound {
		return logDposError(ctx, errors.New("No delegation to redelegate."), req.String())
	} else if err != nil {
		return err
	}

	newLocktimeTier := priorDelegation.LocktimeTier
	newLocktime := priorDelegation.LockTime

	if req.NewLocktimeTier > uint64(newLocktimeTier) {
		state, err := loadState(ctx)
		if err != nil {
			return err
		}

		newLocktimeTier = LocktimeTier(req.NewLocktimeTier)
		tierTime := TierLocktimeMap[newLocktimeTier]
		now := uint64(ctx.Now().Unix())
		remainingTime := state.Params.ElectionCycleLength - (ctx.Now().Unix() - state.LastElectionTime)
		newLocktime := now + tierTime + uint64(remainingTime)

		if newLocktime < now {
			return logDposError(ctx, errors.New("Overflow in set locktime!"), req.String())
		}
	}

	// if req.Amount == nil, it is assumed caller wants to redelegate full delegation
	if req.Amount == nil || priorDelegation.Amount.Value.Cmp(&req.Amount.Value) == 0 {
		priorDelegation.UpdateValidator = req.ValidatorAddress
		priorDelegation.State = REDELEGATING
		priorDelegation.LocktimeTier = newLocktimeTier
		priorDelegation.LockTime = newLocktime
	} else if priorDelegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
		return logDposError(ctx, errors.New("Redelegation amount out of range."), req.String())
	} else {
		// if less than the full amount is being redelegated, create a new
		// delegation for new validator and unbond from former validator
		priorDelegation.State = UNBONDING
		priorDelegation.UpdateAmount = req.Amount
		index, err := GetNextDelegationIndex(ctx, *req.ValidatorAddress, *priorDelegation.Delegator)
		if err != nil {
			return err
		}

		delegation := &Delegation{
			Validator:    req.ValidatorAddress,
			Delegator:    priorDelegation.Delegator,
			Amount:       loom.BigZeroPB(),
			UpdateAmount: req.Amount,
			LocktimeTier: newLocktimeTier,
			LockTime:     newLocktime,
			State:        BONDING,
			Index:        index,
		}
		if err := SetDelegation(ctx, delegation); err != nil {
			return err
		}
	}

	if err := SetDelegation(ctx, priorDelegation); err != nil {
		return err
	}

	return c.emitDelegatorRedelegatesEvent(ctx, delegator.MarshalPB(), req.Amount, req.Referrer)
}

func (c *DPOS) ConsolidateDelegations(ctx contract.Context, req *ConsolidateDelegationsRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS ConsolidateDelegations", "delegator", delegator, "request", req)

	// Unless considation is for the limbo validator, check that the new
	// validator address corresponds to one of the registered candidates
	if req.ValidatorAddress.Local.Compare(limboValidatorAddress.Local) != 0 {
		candidate := GetCandidate(ctx, loom.UnmarshalAddressPB(req.ValidatorAddress))
		// Delegations can only be made to existing candidates
		if candidate == nil {
			return logDposError(ctx, errCandidateNotFound, req.String())
		}
	}

	_, err := consolidateDelegations(ctx, req.ValidatorAddress, delegator.MarshalPB())
	if err != nil {
		return err
	}

	return c.emitDelegatorConsolidatesEvent(ctx, delegator.MarshalPB(), req.ValidatorAddress)
}

// returns the number of delegations which were not consolidated in the event there is no error
func consolidateDelegations(ctx contract.Context, validator, delegator *types.Address) (int, error) {
	// cycle through all delegations and delete those which are BONDED and
	// unlocked while accumulating their amounts
	delegations, err := returnMatchingDelegations(ctx, validator, delegator)
	if err != nil {
		return -1, err
	}

	unconsolidatedDelegations := 0
	totalDelegationAmount := common.BigZero()
	for _, delegation := range delegations {
		if delegation.LockTime > uint64(ctx.Now().Unix()) || delegation.State != BONDED {
			unconsolidatedDelegations++
			continue
		}

		totalDelegationAmount.Add(totalDelegationAmount, &delegation.Amount.Value)

		if err = DeleteDelegation(ctx, delegation); err != nil {
			return -1, err
		}
	}

	index, err := GetNextDelegationIndex(ctx, *validator, *delegator)
	if err != nil {
		return -1, err
	}

	// create new conolidated delegation
	delegation := &Delegation{
		Validator:    validator,
		Delegator:    delegator,
		Amount:       &types.BigUInt{Value: *totalDelegationAmount},
		UpdateAmount: loom.BigZeroPB(),
		LocktimeTier: 0,
		LockTime:     0,
		State:        BONDED,
		Index:        index,
	}
	if err := SetDelegation(ctx, delegation); err != nil {
		return -1, err
	}

	return unconsolidatedDelegations, nil
}

func (c *DPOS) Unbond(ctx contract.Context, req *UnbondRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS Unbond", "delegator", delegator, "request", req)

	delegation, err := GetDelegation(ctx, req.Index, *req.ValidatorAddress, *delegator.MarshalPB())
	if err == contract.ErrNotFound {
		return logDposError(ctx, errors.New(fmt.Sprintf("delegation not found: %s %s", req.ValidatorAddress, delegator.MarshalPB())), req.String())
	} else if err != nil {
		return err
	}

	if delegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
		return logDposError(ctx, errors.New("Unbond amount exceeds delegation amount."), req.String())
	} else if delegation.LockTime > uint64(ctx.Now().Unix()) {
		return logDposError(ctx, errors.New("Delegation currently locked."), req.String())
	} else if delegation.State != BONDED {
		return logDposError(ctx, errors.New("Existing delegation not in BONDED state."), req.String())
	} else {
		delegation.State = UNBONDING
		delegation.UpdateAmount = req.Amount
		SetDelegation(ctx, delegation)
	}

	return c.emitDelegatorUnbondsEvent(ctx, delegator.MarshalPB(), req.Amount)
}

func (c *DPOS) CheckDelegation(ctx contract.StaticContext, req *CheckDelegationRequest) (*CheckDelegationResponse, error) {
	ctx.Logger().Debug("DPOS CheckDelegation", "request", req)

	delegations, err := returnMatchingDelegations(ctx, req.ValidatorAddress, req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	totalDelegationAmount := common.BigZero()
	totalWeightedDelegationAmount := common.BigZero()
	for _, delegation := range delegations {
		totalDelegationAmount.Add(totalDelegationAmount, &delegation.Amount.Value)
		weightedAmount := calculateWeightedDelegationAmount(*delegation)
		totalWeightedDelegationAmount.Add(totalWeightedDelegationAmount, &weightedAmount)
	}

	return &CheckDelegationResponse{Amount: &types.BigUInt{Value: *totalDelegationAmount}, WeightedAmount: &types.BigUInt{Value: *totalWeightedDelegationAmount}, Delegations: delegations}, nil
}

func (c *DPOS) CheckAllDelegations(ctx contract.StaticContext, req *CheckAllDelegationsRequest) (*CheckAllDelegationsResponse, error) {
	ctx.Logger().Debug("DPOS CheckAllDelegations", "request", req)

	if req.DelegatorAddress == nil {
		return nil, logStaticDposError(ctx, errors.New("CheckAllDelegations called with req.DelegatorAddress == nil"), req.String())
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	totalDelegationAmount := common.BigZero()
	totalWeightedDelegationAmount := common.BigZero()
	var delegatorDelegations []*Delegation
	for _, d := range delegations {
		if d.Delegator.Local.Compare(req.DelegatorAddress.Local) != 0 {
			continue
		}

		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		totalDelegationAmount.Add(totalDelegationAmount, &delegation.Amount.Value)
		weightedAmount := calculateWeightedDelegationAmount(*delegation)
		totalWeightedDelegationAmount.Add(totalWeightedDelegationAmount, &weightedAmount)
		delegatorDelegations = append(delegatorDelegations, delegation)
	}

	return &CheckAllDelegationsResponse{Amount: &types.BigUInt{Value: *totalDelegationAmount}, WeightedAmount: &types.BigUInt{Value: *totalWeightedDelegationAmount}, Delegations: delegatorDelegations}, nil
}

// **************************
// CANDIDATE REGISTRATION
// **************************

func (c *DPOS) WhitelistCandidate(ctx contract.Context, req *WhitelistCandidateRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS WhitelistCandidate", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	return c.addCandidateToStatisticList(ctx, req)
}

func (c *DPOS) addCandidateToStatisticList(ctx contract.Context, req *WhitelistCandidateRequest) error {
	_, err := GetStatistic(ctx, loom.UnmarshalAddressPB(req.CandidateAddress))
	if err == contract.ErrNotFound {
		// Creating a ValidatorStatistic entry for candidate with the appropriate
		// lockup period and amount
		SetStatistic(ctx, &ValidatorStatistic{
			Address:           req.CandidateAddress,
			WhitelistAmount:   req.Amount,
			DistributionTotal: loom.BigZeroPB(),
			DelegationTotal:   loom.BigZeroPB(),
			SlashPercentage:   loom.BigZeroPB(),
		})
	} else if err == nil {
		// ValidatorStatistic must not yet exist for a particular candidate in order
		// to be whitelisted
		return logDposError(ctx, errors.New("Cannot whitelist an already whitelisted candidate."), req.String())
	} else {
		return logDposError(ctx, err, req.String())
	}

	// add a 0-value delegation if no others exist; this ensures that an
	// election will be triggered
	if DelegationsCount(ctx) == 0 {
		delegation := &Delegation{
			Validator:    req.CandidateAddress,
			Delegator:    req.CandidateAddress,
			Amount:       loom.BigZeroPB(),
			UpdateAmount: loom.BigZeroPB(),
			LocktimeTier: TierMap[0],
			LockTime:     uint64(ctx.Now().Unix()),
			State:        BONDED,
			Index:        0,
		}
		if err := SetDelegation(ctx, delegation); err != nil {
			return err
		}
	}

	return nil
}

func (c *DPOS) RemoveWhitelistedCandidate(ctx contract.Context, req *RemoveWhitelistedCandidateRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS RemoveWhitelistCandidate", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	statistic, err := GetStatistic(ctx, loom.UnmarshalAddressPB(req.CandidateAddress))
	if err != contract.ErrNotFound && err != nil {
		return err
	}

	if statistic == nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	}
	statistic.WhitelistAmount = loom.BigZeroPB()
	return SetStatistic(ctx, statistic)
}

func (c *DPOS) ChangeWhitelistAmount(ctx contract.Context, req *ChangeWhitelistAmountRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS", "ChangeWhitelistAmount", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())

	}

	statistic, err := GetStatistic(ctx, loom.UnmarshalAddressPB(req.CandidateAddress))
	if err != nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	}

	statistic.WhitelistAmount = req.Amount

	return SetStatistic(ctx, statistic)
}

func (c *DPOS) RegisterCandidate(ctx contract.Context, req *RegisterCandidateRequest) error {
	candidateAddress := ctx.Message().Sender
	ctx.Logger().Info("DPOS RegisterCandidate", "candidate", candidateAddress, "request", req)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	checkAddr := loom.LocalAddressFromPublicKey(req.PubKey)
	if candidateAddress.Local.Compare(checkAddr) != 0 {
		return logDposError(ctx, errors.New("Public key does not match address."), req.String())
	}

	// if candidate record already exists, exit function; candidate record
	// updates are done via the UpdateCandidateRecord function
	cand := candidates.Get(candidateAddress)
	if cand != nil {
		return logDposError(ctx, errCandidateAlreadyRegistered, req.String())
	}

	// Don't check for an err here becuase a nil statistic is expected when
	// a candidate registers for the first time
	statistic, _ := GetStatistic(ctx, candidateAddress)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if (statistic == nil || common.IsZero(statistic.WhitelistAmount.Value)) && common.IsPositive(state.Params.RegistrationRequirement.Value) {
		// A currently unregistered candidate must make a loom token deposit
		// = 'registrationRequirement' in order to run for validator.
		coin, err := loadCoin(ctx)
		if err != nil {
			return err
		}

		dposContractAddress := ctx.ContractAddress()
		err = coin.TransferFrom(candidateAddress, dposContractAddress, &state.Params.RegistrationRequirement.Value)
		if err != nil {
			return err
		}

		// Self-delegate funds for the amount of time specified
		tier := req.GetLocktimeTier()
		if tier > 3 {
			return logDposError(ctx, errors.New("Invalid locktime tier"), req.String())
		}

		locktimeTier := TierMap[tier]
		tierTime := TierLocktimeMap[locktimeTier]
		now := uint64(ctx.Now().Unix())
		lockTime := now + tierTime

		delegation := &Delegation{
			Validator:    candidateAddress.MarshalPB(),
			Delegator:    candidateAddress.MarshalPB(),
			Amount:       loom.BigZeroPB(),
			UpdateAmount: state.Params.RegistrationRequirement,
			LocktimeTier: locktimeTier,
			LockTime:     lockTime,
			State:        BONDING,
			Index:        DELEGATION_START_INDEX,
		}
		if err := SetDelegation(ctx, delegation); err != nil {
			return err
		}
	}

	newCandidate := &Candidate{
		PubKey:      req.PubKey,
		Address:     candidateAddress.MarshalPB(),
		Fee:         req.Fee,
		NewFee:      req.Fee,
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
		State:       REGISTERED,
	}
	candidates.Set(newCandidate)

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateRegistersEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) ChangeFee(ctx contract.Context, req *ChangeCandidateFeeRequest) error {
	ctx.Logger().Info("DPOS ChangeFee", "request", req)

	candidateAddress := ctx.Message().Sender
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return errCandidateNotFound
	}

	if cand.State != REGISTERED {
		return logDposError(ctx, errors.New("Candidate not in REGISTERED state."), req.String())
	}

	cand.NewFee = req.Fee
	cand.State = ABOUT_TO_CHANGE_FEE

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateFeeChangeEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) UpdateCandidateInfo(ctx contract.Context, req *UpdateCandidateInfoRequest) error {
	ctx.Logger().Info("DPOS UpdateCandidateInfo", "request", req)

	candidateAddress := ctx.Message().Sender
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return errCandidateNotFound
	}

	cand.Name = req.Name
	cand.Description = req.Description
	cand.Website = req.Website

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitUpdateCandidateInfoEvent(ctx, candidateAddress.MarshalPB())
}

// When UnregisterCandidate is called, all slashing must be applied to
// delegators. Delegators can be unbonded AFTER SOME WITHDRAWAL DELAY.
// Leaving the validator set mid-election period results in a loss of rewards
// but it should not result in slashing due to downtime.
func (c *DPOS) UnregisterCandidate(ctx contract.Context, req *UnregisterCandidateRequest) error {
	candidateAddress := ctx.Message().Sender
	ctx.Logger().Info("DPOS RemoveWhitelistCandidate", "candidateAddress", candidateAddress, "request", req)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return logDposError(ctx, errCandidateNotFound, req.String())
	} else if cand.State != REGISTERED {
		return logDposError(ctx, errors.New("Candidate not in REGISTERED state."), req.String())
	} else {
		cand.State = UNREGISTERING

		// unbond all validator self-delegations by first consolidating & then unbonding single delegation
		lockedDelegations, err := consolidateDelegations(ctx, candidateAddress.MarshalPB(), candidateAddress.MarshalPB())
		if err != nil {
			return err
		}
		if lockedDelegations != 0 {
			return logDposError(ctx, errors.New("Validator has locked self-delegations."), req.String())
		}

		// After successful consolidation, only one delegation remains at DELEGATION_START_INDEX
		delegation, err := GetDelegation(ctx, DELEGATION_START_INDEX, *candidateAddress.MarshalPB(), *candidateAddress.MarshalPB())
		if err != contract.ErrNotFound && err != nil {
			return err
		}

		// In case that a whitelisted candidate with no self-delegation calls this
		// function, we must check that delegation is not nil
		if delegation != nil && !common.IsZero(delegation.Amount.Value) {
			if delegation.LockTime > uint64(ctx.Now().Unix()) {
				return logDposError(ctx, errors.New("Validator's self-delegation currently locked."), req.String())
			} else if delegation.State != BONDED {
				return logDposError(ctx, errors.New(fmt.Sprintf("Existing delegation not in BONDED state. state: %s", delegation.State)), req.String())
			} else {
				// Once this delegation is unbonded, the total self-delegation
				// amount will be returned to the unregistered validator
				delegation.State = UNBONDING
				delegation.UpdateAmount = &types.BigUInt{Value: delegation.Amount.Value}
				if err := SetDelegation(ctx, delegation); err != nil {
					return err
				}
			}
		}

		statistic, err := GetStatistic(ctx, candidateAddress)
		if err != nil {
			return err
		}

		if err = saveCandidateList(ctx, candidates); err != nil {
			return err
		}

		slashValidatorDelegations(ctx, statistic, candidateAddress)
	}

	return c.emitCandidateUnregistersEvent(ctx, candidateAddress.MarshalPB())
}

func (c *DPOS) ListCandidates(ctx contract.StaticContext, req *ListCandidateRequest) (*ListCandidateResponse, error) {
	ctx.Logger().Debug("DPOS ListCandidates", "request", req)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	candidateStat := make([]*CandidateStatistic, 0)
	for _, candidate := range candidates {
		// Don't check for nil statistic, it will only be nil before the first elections right after a candidate registers
		statistic, err := GetStatistic(ctx, loom.UnmarshalAddressPB(candidate.Address))
		if err != nil && err != contract.ErrNotFound {
			return nil, err
		}

		candidateStat = append(candidateStat, &CandidateStatistic{
			Candidate: candidate,
			Statistic: statistic,
		})
	}

	return &ListCandidateResponse{
		Candidates: candidateStat,
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

	// When there are no token delegations and no statistics (which contain
	// whitelist delegation amounts), quit the function early and leave the
	// validators as they are
	if DelegationsCount(ctx) == 0 {
		return nil
	}

	delegationResults, err := rewardAndSlash(ctx, state)
	if err != nil {
		return err
	}

	validatorCount := int(state.Params.ValidatorCount)
	if len(delegationResults) < validatorCount {
		validatorCount = len(delegationResults)
	}

	validators := make([]*Validator, 0)
	totalValidatorDelegations := common.BigZero()
	for _, res := range delegationResults[:validatorCount] {
		candidate := GetCandidate(ctx, res.ValidatorAddress)
		if candidate != nil && common.IsPositive(res.DelegationTotal) {
			// checking that DelegationTotal is positive ensures ensures that if
			// by accident a negative delegation total is calculated, the chain
			// does not halt due to the error. 0-value delegations are best to
			// exclude for efficiency, though tendermint would ignore 0-powered
			// validators

			var power big.Int
			// making sure that the validator power can fit into a int64
			power.Div(res.DelegationTotal.Int, powerCorrection)
			validatorPower := power.Int64()
			delegationTotal := &types.BigUInt{Value: res.DelegationTotal}
			totalValidatorDelegations.Add(totalValidatorDelegations, &res.DelegationTotal)
			validators = append(validators, &Validator{
				PubKey: candidate.PubKey,
				Power:  validatorPower,
			})

			statistic, _ := GetStatistic(ctx, loom.UnmarshalAddressPB(candidate.Address))
			if statistic == nil {
				statistic = &ValidatorStatistic{
					Address:           res.ValidatorAddress.MarshalPB(),
					PubKey:            candidate.PubKey,
					DistributionTotal: loom.BigZeroPB(),
					DelegationTotal:   delegationTotal,
					SlashPercentage:   loom.BigZeroPB(),
					WhitelistAmount:   loom.BigZeroPB(),
				}
			} else {
				statistic.DelegationTotal = delegationTotal
				// Needed in case pubkey was not set during whitelisting
				statistic.PubKey = candidate.PubKey
			}

			if err = SetStatistic(ctx, statistic); err != nil {
				return err
			}
		}
	}

	// calling `applyPowerCap` ensure that no validator has >28% of the voting
	// power
	state.Validators = applyPowerCap(validators)
	state.LastElectionTime = ctx.Now().Unix()
	state.TotalValidatorDelegations = &types.BigUInt{Value: *totalValidatorDelegations}

	if err = updateCandidateList(ctx); err != nil {
		return err
	}

	ctx.Logger().Debug("DPOS Elect", "Post-Elect State", state)
	if err = saveState(ctx, state); err != nil {
		return err
	}

	return emitElectionEvent(ctx)
}

// `applyPowerCap` ensures that
// 1) no validator has greater than 28% of power
// 2) power total is approx. unchanged as a result of cap
// 3) ordering of validators by power does not change as a result of cap
func applyPowerCap(validators []*Validator) []*Validator {
	// It is impossible to apply a powercap when the number of validators is
	// less than 4
	if len(validators) < 4 {
		return validators
	}

	powerSum := int64(0)
	max := int64(0)
	for _, v := range validators {
		powerSum += v.Power
		if v.Power > max {
			max = v.Power
		}
	}

	limit := float64(0.28)
	maximumIndividualPower := int64(limit * float64(powerSum))

	if max > maximumIndividualPower {
		extraSum := int64(0)
		underCount := 0
		for _, v := range validators {
			if v.Power > maximumIndividualPower {
				extraSum += v.Power - maximumIndividualPower
				v.Power = maximumIndividualPower
			} else {
				underCount++
			}
		}

		underBoost := int64(float64(extraSum) / float64(underCount))

		for _, v := range validators {
			if v.Power < maximumIndividualPower {
				if v.Power+underBoost > maximumIndividualPower {
					v.Power = maximumIndividualPower
				} else {
					v.Power += underBoost
				}
			}
		}
	}

	return validators
}

func (c *DPOS) TimeUntilElection(ctx contract.StaticContext, req *TimeUntilElectionRequest) (*TimeUntilElectionResponse, error) {
	ctx.Logger().Debug("DPOS TimeUntilEleciton", "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	remainingTime := state.Params.ElectionCycleLength - (ctx.Now().Unix() - state.LastElectionTime)
	return &TimeUntilElectionResponse{
		TimeUntilElection: remainingTime,
	}, nil
}

func (c *DPOS) ListValidators(ctx contract.StaticContext, req *ListValidatorsRequest) (*ListValidatorsResponse, error) {
	ctx.Logger().Debug("DPOS ListValidators", "request", req)

	validators, err := ValidatorList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	displayStatistics := make([]*ValidatorStatistic, 0)
	for _, validator := range validators {
		address := loom.Address{Local: loom.LocalAddressFromPublicKey(validator.PubKey)}

		// get validator statistics
		stat, _ := GetStatistic(ctx, address)
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

func (c *DPOS) ListDelegations(ctx contract.StaticContext, req *ListDelegationsRequest) (*ListDelegationsResponse, error) {
	ctx.Logger().Debug("DPOS ListDelegations", "request", req)

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	total := common.BigZero()
	candidateDelegations := make([]*Delegation, 0)
	for _, d := range delegations {
		if d.Validator.Local.Compare(req.Candidate.Local) != 0 {
			continue
		}

		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		candidateDelegations = append(candidateDelegations, delegation)
		total = total.Add(total, &delegation.Amount.Value)
	}

	return &ListDelegationsResponse{
		Delegations:     candidateDelegations,
		DelegationTotal: &types.BigUInt{Value: *total},
	}, nil
}

func (c *DPOS) ListAllDelegations(ctx contract.StaticContext, req *ListAllDelegationsRequest) (*ListAllDelegationsResponse, error) {
	ctx.Logger().Debug("DPOS ListAllDelegations", "request", req)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*ListDelegationsResponse, 0)
	for _, candidate := range candidates {
		response, err := c.ListDelegations(ctx, &ListDelegationsRequest{Candidate: candidate.Address})
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	return &ListAllDelegationsResponse{
		ListResponses: responses,
	}, nil
}

// ***************************
// REWARDS & SLASHING
// ***************************

// only called for validators, never delegators
func SlashInactivity(ctx contract.Context, validatorAddr []byte) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	return slash(ctx, validatorAddr, state.Params.CrashSlashingPercentage.Value)
}

func SlashDoubleSign(ctx contract.Context, validatorAddr []byte) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	return slash(ctx, validatorAddr, state.Params.ByzantineSlashingPercentage.Value)
}

func slash(ctx contract.Context, validatorAddr []byte, slashPercentage loom.BigUInt) error {
	statistic, err := GetStatisticByAddressBytes(ctx, validatorAddr)
	if err != nil {
		return logDposError(ctx, err, "")
	}

	// If slashing percentage is less than current total slash percentage, do
	// not further increase total slash percentage during this election period
	if slashPercentage.Cmp(&statistic.SlashPercentage.Value) < 0 {
		return nil
	}

	updatedAmount := common.BigZero()
	updatedAmount.Add(&statistic.SlashPercentage.Value, &slashPercentage)
	statistic.SlashPercentage = &types.BigUInt{Value: *updatedAmount}

	if err = SetStatistic(ctx, statistic); err != nil {
		return err
	}

	return emitSlashEvent(ctx, statistic.Address, slashPercentage)
}

// Returns the total amount of tokens which have been distributed to delegators
// and validators as rewards
func (c *DPOS) CheckRewards(ctx contract.StaticContext, req *CheckRewardsRequest) (*CheckRewardsResponse, error) {
	ctx.Logger().Debug("DPOS CheckRewards", "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	return &CheckRewardsResponse{TotalRewardDistribution: state.TotalRewardDistribution}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&DPOS{})

// UTILITIES

func loadCoin(ctx contract.Context) (*ERC20, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	coinAddr := loom.UnmarshalAddressPB(state.Params.CoinContractAddress)
	return &ERC20{
		Context:         ctx,
		ContractAddress: coinAddr,
	}, nil
}

// rewards & slashes are calculated along with former delegation totals
// rewards are distributed to validators based on fee
// rewards distribution amounts are prepared for delegators
func rewardAndSlash(ctx contract.Context, state *State) ([]*DelegationResult, error) {
	formerValidatorTotals := make(map[string]loom.BigUInt)
	delegatorRewards := make(map[string]*loom.BigUInt)
	for _, validator := range state.Validators {
		// get candidate record to lookup fee
		candidate := GetCandidateByPubKey(ctx, validator.PubKey)

		if candidate == nil {
			ctx.Logger().Info("Attempted to reward validator no longer on candidates list.", "validator", validator)
			continue
		}

		candidateAddress := loom.UnmarshalAddressPB(candidate.Address)
		validatorKey := candidateAddress.String()
		//get validator statistics
		statistic, _ := GetStatistic(ctx, candidateAddress)

		if statistic == nil {
			delegatorRewards[validatorKey] = common.BigZero()
			formerValidatorTotals[validatorKey] = *common.BigZero()
		} else {
			// If a validator's SlashPercentage is 0, the validator is
			// rewarded for avoiding faults during the last slashing period
			if common.IsZero(statistic.SlashPercentage.Value) {
				rewardValidator(statistic, state.Params, state.TotalValidatorDelegations.Value)

				validatorShare := CalculateFraction(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, statistic.DistributionTotal.Value)

				// increase validator's delegation
				IncreaseRewardDelegation(ctx, candidate.Address, candidate.Address, validatorShare)

				// delegatorsShare is the amount to all delegators in proportion
				// to the amount that they've delegatored
				delegatorsShare := common.BigZero()
				delegatorsShare.Sub(&statistic.DistributionTotal.Value, &validatorShare)
				delegatorRewards[validatorKey] = delegatorsShare

				// If a validator has some non-zero WhitelistAmount,
				// calculate the validator's reward based on whitelist amount
				if !common.IsZero(statistic.WhitelistAmount.Value) {
					whitelistDistribution := calculateShare(statistic.WhitelistAmount.Value, statistic.DelegationTotal.Value, *delegatorsShare)
					// increase a delegator's distribution
					IncreaseRewardDelegation(ctx, candidate.Address, candidate.Address, whitelistDistribution)
				}
			} else {
				slashValidatorDelegations(ctx, statistic, candidateAddress)
			}

			// Zeroing out validator's distribution total since it will be transfered
			// to the distributions storage during this `Elect` call.
			// Validators and Delegators both can claim their rewards in the
			// same way when this is true.
			state.TotalRewardDistribution.Value.Add(&state.TotalRewardDistribution.Value, &statistic.DistributionTotal.Value)
			statistic.DistributionTotal = loom.BigZeroPB()
			formerValidatorTotals[validatorKey] = statistic.DelegationTotal.Value
		}
	}

	newDelegationTotals, err := distributeDelegatorRewards(ctx, formerValidatorTotals, delegatorRewards)
	if err != nil {
		return nil, err
	}

	delegationResults := make([]*DelegationResult, 0, len(newDelegationTotals))
	for validator := range newDelegationTotals {
		delegationResults = append(delegationResults, &DelegationResult{
			ValidatorAddress: loom.MustParseAddress(validator),
			DelegationTotal:  *newDelegationTotals[validator],
		})
	}
	sort.Sort(byDelegationTotal(delegationResults))

	return delegationResults, nil
}

func rewardValidator(statistic *ValidatorStatistic, params *Params, totalValidatorDelegations loom.BigUInt) {
	// if there is no slashing to be applied, reward validator
	cycleSeconds := params.ElectionCycleLength
	reward := CalculateFraction(blockRewardPercentage, statistic.DelegationTotal.Value)

	// if totalValidator Delegations are high enough to make simple reward
	// calculations result in more rewards given out than the value of `MaxYearlyReward`,
	// scale the rewards appropriately
	yearlyRewardTotal := CalculateFraction(blockRewardPercentage, totalValidatorDelegations)
	if yearlyRewardTotal.Cmp(&params.MaxYearlyReward.Value) > 0 {
		reward.Mul(&reward, &params.MaxYearlyReward.Value)
		reward.Div(&reward, &yearlyRewardTotal)
	}

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

func slashValidatorDelegations(ctx contract.Context, statistic *ValidatorStatistic, validatorAddress loom.Address) error {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	// these delegation totals will be added back up again when we calculate new delegation totals below
	for _, d := range delegations {
		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return err
		}
		// check the it's a delegation that belongs to the validator
		if delegation.Validator.Local.Compare(validatorAddress.Local) == 0 && !common.IsZero(statistic.SlashPercentage.Value) {
			toSlash := CalculateFraction(statistic.SlashPercentage.Value, delegation.Amount.Value)
			updatedAmount := common.BigZero()
			updatedAmount.Sub(&delegation.Amount.Value, &toSlash)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
			if err := SetDelegation(ctx, delegation); err != nil {
				return err
			}
		}
	}

	// Slash a whitelisted candidate's whitelist amount. This doesn't affect how
	// much the validator gets back from token timelock, but will decrease the
	// validator's delegation total & thus his ability to earn rewards
	if !common.IsZero(statistic.WhitelistAmount.Value) {
		toSlash := CalculateFraction(statistic.SlashPercentage.Value, statistic.WhitelistAmount.Value)
		updatedAmount := common.BigZero()
		updatedAmount.Sub(&statistic.WhitelistAmount.Value, &toSlash)
		statistic.WhitelistAmount = &types.BigUInt{Value: *updatedAmount}
	}

	// reset slash total
	statistic.SlashPercentage = loom.BigZeroPB()

	return nil
}

// This function has three goals 1) distribute a validator's rewards to each of
// the delegators, 2) finalize the bonding process for any delegations recieved
// during the last election period (delegate & unbond calls) and 3) calculate
// the new delegation totals.
func distributeDelegatorRewards(ctx contract.Context, formerValidatorTotals map[string]loom.BigUInt, delegatorRewards map[string]*loom.BigUInt) (map[string]*loom.BigUInt, error) {
	newDelegationTotals := make(map[string]*loom.BigUInt)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	// initialize delegation totals with whitelist amounts
	for _, candidate := range candidates {
		statistic, _ := GetStatistic(ctx, loom.UnmarshalAddressPB(candidate.Address))

		if statistic != nil && statistic.WhitelistAmount != nil && !common.IsZero(statistic.WhitelistAmount.Value) {
			validatorKey := loom.UnmarshalAddressPB(statistic.Address).String()
			// WhitelistAmount is not weighted because it is assumed Oracle
			// added appropriate bonus during registration
			newDelegationTotals[validatorKey] = &statistic.WhitelistAmount.Value
		}
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	for _, d := range delegations {
		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		validatorKey := loom.UnmarshalAddressPB(delegation.Validator).String()

		// Do do distribute rewards to delegators of the Limbo validators
		if delegation.Validator.Local.Compare(limboValidatorAddress.Local) != 0 {
			// allocating validator distributions to delegators
			// based on former validator delegation totals
			delegationTotal := formerValidatorTotals[validatorKey]
			rewardsTotal := delegatorRewards[validatorKey]
			if rewardsTotal != nil {
				weightedDelegation := calculateWeightedDelegationAmount(*delegation)
				delegatorDistribution := calculateShare(weightedDelegation, delegationTotal, *rewardsTotal)
				// increase a delegator's distribution
				IncreaseRewardDelegation(ctx, delegation.Validator, delegation.Delegator, delegatorDistribution)
			}
		}

		updatedAmount := common.BigZero()
		if delegation.State == BONDING {
			updatedAmount.Add(&delegation.Amount.Value, &delegation.UpdateAmount.Value)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
		} else if delegation.State == UNBONDING {
			updatedAmount.Sub(&delegation.Amount.Value, &delegation.UpdateAmount.Value)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
			coin, err := loadCoin(ctx)
			if err != nil {
				return nil, err
			}
			err = coin.Transfer(loom.UnmarshalAddressPB(delegation.Delegator), &delegation.UpdateAmount.Value)
			if err != nil {
				return nil, err
			}
		} else if delegation.State == REDELEGATING {
			if err = DeleteDelegation(ctx, delegation); err != nil {
				return nil, err
			}
			delegation.Validator = delegation.UpdateValidator
			validatorKey = loom.UnmarshalAddressPB(delegation.Validator).String()
		}

		// Delete any delegation whose full amount has been unbonded. In all
		// other cases, update the delegation state to BONDED and reset its
		// UpdateAmount
		if common.IsZero(delegation.Amount.Value) && delegation.State == UNBONDING {
			if err := DeleteDelegation(ctx, delegation); err != nil {
				return nil, err
			}
		} else {
			// After a delegation update, zero out UpdateAmount
			delegation.UpdateAmount = loom.BigZeroPB()
			delegation.State = BONDED

			resetDelegationIfExpired(ctx, delegation)
			if err := SetDelegation(ctx, delegation); err != nil {
				return nil, err
			}
		}

		// Calculate delegation total of the Limbo validator
		if delegation.Validator.Local.Compare(limboValidatorAddress.Local) != 0 {
			newTotal := common.BigZero()
			weightedDelegation := calculateWeightedDelegationAmount(*delegation)
			newTotal.Add(newTotal, &weightedDelegation)
			if newDelegationTotals[validatorKey] != nil {
				newTotal.Add(newTotal, newDelegationTotals[validatorKey])
			}
			newDelegationTotals[validatorKey] = newTotal
		}
	}

	return newDelegationTotals, nil
}

// Reset a delegation's tier to 0 if it's locktime has expired
func resetDelegationIfExpired(ctx contract.Context, delegation *Delegation) {
	now := uint64(ctx.Now().Unix())
	if delegation.LocktimeTier != TIER_ZERO && delegation.LockTime < now {
		delegation.LocktimeTier = TIER_ZERO
	}
}

func returnMatchingDelegations(ctx contract.StaticContext, validator, delegator *types.Address) ([]*Delegation, error) {
	if validator == nil {
		return nil, errors.New("request made with req.ValidatorAddress == nil")
	}
	if delegator == nil {
		return nil, errors.New("request made with req.DelegatorAddress == nil")
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	var matchingDelegations []*Delegation
	for _, d := range delegations {
		if d.Delegator.Local.Compare(delegator.Local) != 0 || d.Validator.Local.Compare(validator.Local) != 0 {
			continue
		}

		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		matchingDelegations = append(matchingDelegations, delegation)
	}

	return matchingDelegations, nil
}

func (c *DPOS) CheckRewardDelegation(ctx contract.StaticContext, req *CheckRewardDelegationRequest) (*CheckRewardDelegationResponse, error) {
	delegator := ctx.Message().Sender
	ctx.Logger().Debug("DPOS CheckRewardDelegation", "delegator", delegator, "request", req)

	if req.ValidatorAddress == nil {
		return nil, logStaticDposError(ctx, errors.New("CheckRewardDelegation called with req.ValdiatorAddress == nil"), req.String())
	}

	delegation, err := GetDelegation(ctx, REWARD_DELEGATION_INDEX, *req.ValidatorAddress, *delegator.MarshalPB())
	if err == contract.ErrNotFound {
		delegation = &Delegation{
			Validator:    req.ValidatorAddress,
			Delegator:    delegator.MarshalPB(),
			Amount:       loom.BigZeroPB(),
			UpdateAmount: loom.BigZeroPB(),
			LocktimeTier: TierMap[0],
			LockTime:     0,
			State:        BONDED,
			Index:        REWARD_DELEGATION_INDEX,
		}
	} else if err != nil {
		return nil, err
	}

	resp := &CheckRewardDelegationResponse{Delegation: delegation}

	return resp, nil
}

func (c *DPOS) GetState(ctx contract.StaticContext, req *GetStateRequest) (*GetStateResponse, error) {
	ctx.Logger().Debug("DPOS", "GetState", "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	return &GetStateResponse{State: state}, nil
}

// *************************
// ORACLE METHODS
// *************************

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
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errors.New("[ProcessRequestBatch] only oracle is authorized to call ProcessRequestBatch"), req.String())
	}

	if req.Batch == nil || len(req.Batch) == 0 {
		return logDposError(ctx, errors.New("[ProcessRequestBatch] invalid Request, no batch request found"), req.String())
	}

	tally, err := loadRequestBatchTally(ctx)
	if err != nil {
		return err
	}

	lastRequest := req.Batch[len(req.Batch)-1]
	if isRequestAlreadySeen(lastRequest.Meta, tally) {
		return logDposError(ctx, errors.New("[ProcessRequestBatch] invalid Request, all events has been already seen"), req.String())
	}

loop:
	for _, request := range req.Batch {
		switch payload := request.Payload.(type) {
		case *dtypes.BatchRequest_WhitelistCandidate:
			if isRequestAlreadySeen(request.Meta, tally) {
				break
			}

			if err = c.addCandidateToStatisticList(ctx, payload.WhitelistCandidate); err != nil {
				break loop
			}

			tally.LastSeenBlockNumber = request.Meta.BlockNumber
			tally.LastSeenTxIndex = request.Meta.TxIndex
			tally.LastSeenLogIndex = request.Meta.LogIndex
		default:
			err = logDposError(ctx, errors.New("unsupported type of request in request batch"), req.String())
		}
	}

	if err != nil {
		return logDposError(ctx, errors.New(fmt.Sprintf("[ProcessRequestBatch] unable to consume one or more request, error: %v", err)), req.String())
	}

	if err = saveRequestBatchTally(ctx, tally); err != nil {
		return logDposError(ctx, errors.New(fmt.Sprintf("[ProcessRequestBatch] unable to save request tally, error: %v", err)), req.String())
	}

	return nil
}

func (c *DPOS) SetElectionCycle(ctx contract.Context, req *SetElectionCycleRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS SetElectionCycle", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.ElectionCycleLength = req.ElectionCycle

	return saveState(ctx, state)
}

func (c *DPOS) SetMaxYearlyReward(ctx contract.Context, req *SetMaxYearlyRewardRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS SetMaxYearlyReward", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.MaxYearlyReward = req.MaxYearlyReward

	return saveState(ctx, state)
}

func (c *DPOS) SetRegistrationRequirement(ctx contract.Context, req *SetRegistrationRequirementRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS SetRegistrationRequirement", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.RegistrationRequirement = req.RegistrationRequirement

	return saveState(ctx, state)
}

func (c *DPOS) SetValidatorCount(ctx contract.Context, req *SetValidatorCountRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS SetValidatorCount", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	return saveState(ctx, state)
}

func (c *DPOS) SetOracleAddress(ctx contract.Context, req *SetOracleAddressRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS SetOracleAddress", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.OracleAddress = req.OracleAddress

	return saveState(ctx, state)
}

func (c *DPOS) SetSlashingPercentages(ctx contract.Context, req *SetSlashingPercentagesRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS SetSlashingPercentage", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.CrashSlashingPercentage = req.CrashSlashingPercentage
	state.Params.ByzantineSlashingPercentage = req.ByzantineSlashingPercentage

	return saveState(ctx, state)
}

// ***************************************
// STATE-CHANGE LOGGING EVENTS
// ***************************************

func emitElectionEvent(ctx contract.Context) error {
	marshalled, err := proto.Marshal(&DposElectionEvent{
		BlockNumber: uint64(ctx.Block().Height),
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, ElectionEventTopic)
	return nil
}

func emitSlashEvent(ctx contract.Context, validator *types.Address, slashPercentage loom.BigUInt) error {
	marshalled, err := proto.Marshal(&DposSlashEvent{
		Validator:       validator,
		SlashPercentage: &types.BigUInt{Value: slashPercentage},
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, SlashEventTopic)
	return nil
}

func (c *DPOS) emitCandidateRegistersEvent(ctx contract.Context, candidate *types.Address, fee uint64) error {
	marshalled, err := proto.Marshal(&DposCandidateRegistersEvent{
		Address: candidate,
		Fee:     fee,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, CandidateRegistersEventTopic)
	return nil
}

func (c *DPOS) emitCandidateUnregistersEvent(ctx contract.Context, candidate *types.Address) error {
	marshalled, err := proto.Marshal(&DposCandidateUnregistersEvent{
		Address: candidate,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, CandidateUnregistersEventTopic)
	return nil
}

func (c *DPOS) emitCandidateFeeChangeEvent(ctx contract.Context, candidate *types.Address, fee uint64) error {
	marshalled, err := proto.Marshal(&DposCandidateFeeChangeEvent{
		Address: candidate,
		NewFee:  fee,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, CandidateFeeChangeEventTopic)
	return nil
}

func (c *DPOS) emitUpdateCandidateInfoEvent(ctx contract.Context, candidate *types.Address) error {
	marshalled, err := proto.Marshal(&DposUpdateCandidateInfoEvent{
		Address: candidate,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, UpdateCandidateInfoEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorDelegatesEvent(ctx contract.Context, delegator *types.Address, amount *types.BigUInt, referrer string) error {
	marshalled, err := proto.Marshal(&DposDelegatorDelegatesEvent{
		Address:  delegator,
		Amount:   amount,
		Referrer: referrer,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorDelegatesEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorRedelegatesEvent(ctx contract.Context, delegator *types.Address, amount *types.BigUInt, referrer string) error {
	marshalled, err := proto.Marshal(&DposDelegatorRedelegatesEvent{
		Address:  delegator,
		Amount:   amount,
		Referrer: referrer,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorRedelegatesEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorConsolidatesEvent(ctx contract.Context, delegator, validator *types.Address) error {
	marshalled, err := proto.Marshal(&DposDelegatorConsolidatesEvent{
		Address:   delegator,
		Validator: validator,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorConsolidatesEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorUnbondsEvent(ctx contract.Context, delegator *types.Address, amount *types.BigUInt) error {
	marshalled, err := proto.Marshal(&DposDelegatorUnbondsEvent{
		Address: delegator,
		Amount:  amount,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorUnbondsEventTopic)
	return nil
}

// ***************************
// MIGRATION FUNCTIONS
// ***************************

// In case of migration from v2, this should be called in place of Init() and it
// should be the first call made to the dposv3 Contract. The call should be made
// from within the `Dump` function of the dposv2 contract.
func Initialize(ctx contract.Context, initState *InitializationState) error {
	ctx.Logger().Info("DPOSv3 Initialize")
	sender := ctx.Message().Sender

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, initState.String())
	}

	// set new State
	if err := saveState(ctx, initState.State); err != nil {
		return err
	}

	// set new Candidates
	if err := saveCandidateList(ctx, initState.Candidates); err != nil {
		return err
	}

	// set new Delegations
	for _, delegation := range initState.Delegations {
		if err := SetDelegation(ctx, delegation); err != nil {
			return err
		}
	}

	// set new Statistics
	for _, statistic := range initState.Statistics {
		if err := SetStatistic(ctx, statistic); err != nil {
			return err
		}
	}

	return nil
}
