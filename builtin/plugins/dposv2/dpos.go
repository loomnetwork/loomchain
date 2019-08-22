package dposv2

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"

	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/loomnetwork/loomchain/feature"
)

const (
	defaultRegistrationRequirement = 1250000
	defaultMaxYearlyReward         = 60000000
	tokenDecimals                  = 18
	yearSeconds                    = int64(60 * 60 * 24 * 365)
	BONDING                        = dtypes.DelegationV2_BONDING
	BONDED                         = dtypes.DelegationV2_BONDED
	UNBONDING                      = dtypes.DelegationV2_UNBONDING
	REDELEGATING                   = dtypes.DelegationV2_REDELEGATING
	TIER_ZERO                      = dtypes.DelegationV2_TIER_ZERO
	TIER_ONE                       = dtypes.DelegationV2_TIER_ONE
	TIER_TWO                       = dtypes.DelegationV2_TIER_TWO
	TIER_THREE                     = dtypes.DelegationV2_TIER_THREE
	feeChangeDelay                 = 2

	ElectionEventTopic               = "dpos:election"
	SlashEventTopic                  = "dpos:slash"
	CandidateRegistersEventTopic     = "dpos:candidateregisters"
	CandidateUnregistersEventTopic   = "dpos:candidateunregisters"
	CandidateFeeChangeEventTopic     = "dpos:candidatefeechange"
	UpdateCandidateInfoEventTopic    = "dpos:updatecandidateinfo"
	DelegatorDelegatesEventTopic     = "dpos:delegatordelegates"
	DelegatorRedelegatesEventTopic   = "dpos:delegatorredelegates"
	DelegatorUnbondsEventTopic       = "dpos:delegatorunbonds"
	DelegatorClaimsRewardsEventTopic = "dpos:delegatorclaimsrewards"
)

var (
	secondsInYear = loom.BigUInt{big.NewInt(yearSeconds)}

	blockRewardPercentage     = loom.BigUInt{big.NewInt(500)}
	doubleSignSlashPercentage = loom.BigUInt{big.NewInt(500)}
	inactivitySlashPercentage = loom.BigUInt{big.NewInt(100)}
	limboValidatorAddress     = loom.MustParseAddress("limbo:0x0000000000000000000000000000000000000000")
	powerCorrection           = big.NewInt(1000000000)
	errCandidateNotFound      = errors.New("Candidate record not found.")
	errValidatorNotFound      = errors.New("Validator record not found.")
	errDistributionNotFound   = errors.New("Distribution record not found.")
	errOnlyOracle             = errors.New("Function can only be called with oracle address.")
	errContractDisabled       = errors.New("Contract disabled after migration to DPOSv3.")
)

type (
	InitRequest                       = dtypes.DPOSInitRequestV2
	DelegateRequest                   = dtypes.DelegateRequestV2
	RedelegateRequest                 = dtypes.RedelegateRequestV2
	WhitelistCandidateRequest         = dtypes.WhitelistCandidateRequestV2
	RemoveWhitelistedCandidateRequest = dtypes.RemoveWhitelistedCandidateRequestV2
	ChangeWhitelistAmountRequest      = dtypes.ChangeWhitelistAmountRequestV2
	DelegationState                   = dtypes.DelegationV2_DelegationState
	LocktimeTier                      = dtypes.DelegationV2_LocktimeTier
	UnbondRequest                     = dtypes.UnbondRequestV2
	ClaimDistributionRequest          = dtypes.ClaimDistributionRequestV2
	ClaimDistributionResponse         = dtypes.ClaimDistributionResponseV2
	CheckAllDelegationsRequest        = dtypes.CheckAllDelegationsRequest
	CheckAllDelegationsResponse       = dtypes.CheckAllDelegationsResponse
	CheckDelegationRequest            = dtypes.CheckDelegationRequestV2
	CheckDelegationResponse           = dtypes.CheckDelegationResponseV2
	TotalDelegationRequest            = dtypes.TotalDelegationRequest
	TotalDelegationResponse           = dtypes.TotalDelegationResponse
	CheckRewardsRequest               = dtypes.CheckRewardsRequest
	CheckRewardsResponse              = dtypes.CheckRewardsResponse
	CheckDistributionRequest          = dtypes.CheckDistributionRequest
	CheckDistributionResponse         = dtypes.CheckDistributionResponse
	TimeUntilElectionRequest          = dtypes.TimeUntilElectionRequest
	TimeUntilElectionResponse         = dtypes.TimeUntilElectionResponse
	RegisterCandidateRequest          = dtypes.RegisterCandidateRequestV2
	ChangeCandidateFeeRequest         = dtypes.ChangeCandidateFeeRequest
	UpdateCandidateInfoRequest        = dtypes.UpdateCandidateInfoRequest
	UnregisterCandidateRequest        = dtypes.UnregisterCandidateRequestV2
	ListCandidateRequest              = dtypes.ListCandidateRequestV2
	ListCandidateResponse             = dtypes.ListCandidateResponseV2
	ListValidatorsRequest             = dtypes.ListValidatorsRequestV2
	ListValidatorsResponse            = dtypes.ListValidatorsResponseV2
	ListDelegationsRequest            = dtypes.ListDelegationsRequest
	ListDelegationsResponse           = dtypes.ListDelegationsResponse
	ListAllDelegationsRequest         = dtypes.ListAllDelegationsRequest
	ListAllDelegationsResponse        = dtypes.ListAllDelegationsResponse
	SetElectionCycleRequest           = dtypes.SetElectionCycleRequestV2
	SetMaxYearlyRewardRequest         = dtypes.SetMaxYearlyRewardRequestV2
	SetRegistrationRequirementRequest = dtypes.SetRegistrationRequirementRequestV2
	SetValidatorCountRequest          = dtypes.SetValidatorCountRequestV2
	SetOracleAddressRequest           = dtypes.SetOracleAddressRequestV2
	SetSlashingPercentagesRequest     = dtypes.SetSlashingPercentagesRequestV2
	Candidate                         = dtypes.CandidateV2
	Delegation                        = dtypes.DelegationV2
	Distribution                      = dtypes.DistributionV2
	ValidatorStatistic                = dtypes.ValidatorStatisticV2
	Validator                         = types.Validator
	State                             = dtypes.StateV2
	StateDump                         = dtypes.StateDumpV2
	Params                            = dtypes.ParamsV2
	GetStateRequest                   = dtypes.GetStateRequest
	GetStateResponse                  = dtypes.GetStateResponse
	ViewStateDumpRequest              = dtypes.ViewStateDumpRequest
	ViewStateDumpResponse             = dtypes.ViewStateDumpResponse
	GetDistributionsRequest           = dtypes.GetDistributionsRequest
	GetDistributionsResponse          = dtypes.GetDistributionsResponse

	DposElectionEvent               = dtypes.DposElectionEvent
	DposSlashEvent                  = dtypes.DposSlashEvent
	DposCandidateRegistersEvent     = dtypes.DposCandidateRegistersEvent
	DposCandidateUnregistersEvent   = dtypes.DposCandidateUnregistersEvent
	DposCandidateFeeChangeEvent     = dtypes.DposCandidateFeeChangeEvent
	DposUpdateCandidateInfoEvent    = dtypes.DposUpdateCandidateInfoEvent
	DposDelegatorDelegatesEvent     = dtypes.DposDelegatorDelegatesEvent
	DposDelegatorRedelegatesEvent   = dtypes.DposDelegatorRedelegatesEvent
	DposDelegatorUnbondsEvent       = dtypes.DposDelegatorUnbondsEvent
	DposDelegatorClaimsRewardsEvent = dtypes.DposDelegatorClaimsRewardsEvent

	RequestBatch                = dtypes.RequestBatchV2
	RequestBatchTally           = dtypes.RequestBatchTallyV2
	BatchRequest                = dtypes.BatchRequestV2
	BatchRequestMeta            = dtypes.BatchRequestMetaV2
	GetRequestBatchTallyRequest = dtypes.GetRequestBatchTallyRequestV2

	ChangeWhitelistLockTimeTierRequest = dtypes.ChangeWhitelistLockTimeTierRequestV2
)

type DPOS struct {
}

func (c *DPOS) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "dposV2",
		Version: "2.0.0",
	}, nil
}

// TODO is there a risk of this being called multiple times?
func (c *DPOS) Init(ctx contract.Context, req *InitRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS Delegate", "delegator", delegator, "request", req)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}
	cand := candidates.Get(loom.UnmarshalAddressPB(req.ValidatorAddress))
	// Delegations can only be made to existing candidates
	if cand == nil {
		return logDposError(ctx, errCandidateNotFound, req.String())
	}
	if req.Amount == nil || !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Must Delegate a positive number of tokens."), req.String())
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	coin := loadCoin(ctx, state.Params)

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
			return logDposError(ctx, errors.New("Existing delegation not in BONDED state."), req.String())
		}
		amount = priorDelegation.Amount
	} else {
		amount = loom.BigZeroPB()
	}

	// Extend locktime by the prior delegation's locktime if it exists
	var locktimeTier LocktimeTier
	switch req.GetLocktimeTier() {
	case 0:
		locktimeTier = TierMap[0]
	case 1:
		locktimeTier = TierMap[1]
	case 2:
		locktimeTier = TierMap[2]
	case 3:
		locktimeTier = TierMap[3]
	default:
		return logDposError(ctx, errors.New("Invalid delegation tier"), req.String())
	}

	now := uint64(ctx.Now().Unix())
	var tierTime uint64
	// If there was no prior delegation, or if the user is supplying a bigger locktime
	if priorDelegation == nil || locktimeTier >= priorDelegation.LocktimeTier {
		if !v2_1 {
			tierTime = calculateTierLocktime(locktimeTier, uint64(state.Params.ElectionCycleLength))
		} else {
			tierTime = TierLocktimeMap[locktimeTier]
		}
	} else {
		if !v2_1 {
			tierTime = calculateTierLocktime(priorDelegation.LocktimeTier, uint64(state.Params.ElectionCycleLength))
		} else {
			tierTime = TierLocktimeMap[priorDelegation.LocktimeTier]
		}
	}
	lockTime := now + tierTime

	if lockTime < now {
		return logDposError(ctx, errors.New("Overflow in set locktime!"), req.String())
	}

	delegation := &Delegation{
		Validator:    req.ValidatorAddress,
		Delegator:    delegator.MarshalPB(),
		Amount:       amount,
		UpdateAmount: req.Amount,
		Height:       uint64(ctx.Block().Height),
		// delegations are locked up for a minimum of an election period
		// from the time of the latest delegation
		LocktimeTier: locktimeTier,
		LockTime:     lockTime,
		State:        BONDING,
	}
	delegations.Set(delegation)

	if err = saveDelegationList(ctx, delegations); err != nil {
		return err
	}

	return c.emitDelegatorDelegatesEvent(ctx, delegator.MarshalPB(), req.Amount, req.Referrer)
}

func (c *DPOS) Redelegate(ctx contract.Context, req *RedelegateRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
		candidates, err := loadCandidateList(ctx)
		if err != nil {
			return err
		}
		candidate := candidates.Get(loom.UnmarshalAddressPB(req.ValidatorAddress))
		// Delegations can only be made to existing candidates
		if candidate == nil {
			return logDposError(ctx, errCandidateNotFound, req.String())
		}
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	priorDelegation := delegations.Get(*req.FormerValidatorAddress, *delegator.MarshalPB())

	if priorDelegation == nil {
		return logDposError(ctx, errors.New("No delegation to redelegate."), req.String())
	}

	// if req.Amount == nil, it is assumed caller wants to redelegate full delegation
	if req.Amount == nil || priorDelegation.Amount.Value.Cmp(&req.Amount.Value) == 0 {
		priorDelegation.UpdateValidator = req.ValidatorAddress
		priorDelegation.State = REDELEGATING
	} else if priorDelegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
		return logDposError(ctx, errors.New("Redelegation amount out of range."), req.String())
	} else {
		// if less than the full amount is being redelegated, create a new
		// delegation for new validator and unbond from former validator
		priorDelegation.State = UNBONDING
		priorDelegation.UpdateAmount = req.Amount

		delegation := &Delegation{
			Validator:    req.ValidatorAddress,
			Delegator:    priorDelegation.Delegator,
			Amount:       loom.BigZeroPB(),
			UpdateAmount: req.Amount,
			Height:       uint64(ctx.Block().Height),
			LocktimeTier: priorDelegation.LocktimeTier,
			LockTime:     priorDelegation.LockTime,
			State:        BONDING,
		}
		delegations.Set(delegation)
	}

	if err = saveDelegationList(ctx, delegations); err != nil {
		return err
	}

	return c.emitDelegatorRedelegatesEvent(ctx, delegator.MarshalPB(), req.Amount, req.Referrer)
}

func (c *DPOS) Delegate2(ctx contract.Context, req *DelegateRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS Delegate2", "delegator", delegator, "request", req)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

	// Ensure validator address is set properly in v2.1
	if v2_1 {
		if req.ValidatorAddress == nil {
			return logDposError(ctx, errors.New("Delegate2 called with req.ValidatorAddress == nil"), req.String())
		}

		if req.ValidatorAddress.ChainId != ctx.Block().ChainID {
			return logDposError(ctx, errors.New("Delegate2 called with invalid chainId for req.ValidatorAddress"), req.String())
		}
	}

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}
	cand := candidates.Get(loom.UnmarshalAddressPB(req.ValidatorAddress))
	// Delegations can only be made to existing candidates
	if cand == nil {
		return logDposError(ctx, errCandidateNotFound, req.String())
	}
	if req.Amount == nil || !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Must Delegate a positive number of tokens."), req.String())
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	coin := loadCoin(ctx, state.Params)

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
			return logDposError(ctx, errors.New("Existing delegation not in BONDED state."), req.String())
		}
		amount = priorDelegation.Amount
	} else {
		amount = loom.BigZeroPB()
	}

	tierNumber := req.GetLocktimeTier()
	if tierNumber < 0 || tierNumber > 3 {
		return logDposError(ctx, errors.New("Invalid delegation tier"), req.String())
	}

	// If was a prior delegation and the user is supplying a smaller locktime
	// extend the locktime by the prior lockup period
	locktimeTier := TierMap[tierNumber]
	if priorDelegation != nil && locktimeTier < priorDelegation.LocktimeTier {
		locktimeTier = priorDelegation.LocktimeTier
	}

	var tierTime uint64
	if !v2_1 {
		tierTime = calculateTierLocktime(locktimeTier, uint64(state.Params.ElectionCycleLength))
	} else {
		tierTime = TierLocktimeMap[locktimeTier]
	}
	now := uint64(ctx.Now().Unix())
	lockTime := now + tierTime

	if lockTime < now {
		return logDposError(ctx, errors.New("Overflow in set locktime!"), req.String())
	}

	delegation := &Delegation{
		Validator:    req.ValidatorAddress,
		Delegator:    delegator.MarshalPB(),
		Amount:       amount,
		UpdateAmount: req.Amount,
		Height:       uint64(ctx.Block().Height),
		// delegations are locked up for a minimum of an election period
		// from the time of the latest delegation
		LocktimeTier: locktimeTier,
		LockTime:     lockTime,
		State:        BONDING,
	}
	delegations.Set2(delegation)

	if err = saveDelegationList(ctx, delegations); err != nil {
		return err
	}

	return c.emitDelegatorDelegatesEvent(ctx, delegator.MarshalPB(), req.Amount, req.Referrer)
}

func (c *DPOS) Unbond(ctx contract.Context, req *UnbondRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOS Unbond", "delegator", delegator, "request", req)

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	delegation := delegations.Get(*req.ValidatorAddress, *delegator.MarshalPB())

	if delegation == nil {
		return logDposError(ctx, errors.New(fmt.Sprintf("delegation not found: %s %s", req.ValidatorAddress, delegator.MarshalPB())), req.String())
	} else {
		if delegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
			return logDposError(ctx, errors.New("Unbond amount exceeds delegation amount."), req.String())
		} else if delegation.LockTime > uint64(ctx.Now().Unix()) {
			return logDposError(ctx, errors.New("Delegation currently locked."), req.String())
		} else if delegation.State != BONDED {
			return logDposError(ctx, errors.New("Existing delegation not in BONDED state."), req.String())
		} else {
			delegation.State = UNBONDING
			delegation.UpdateAmount = req.Amount
			delegations.Set(delegation)
		}
	}

	if err = saveDelegationList(ctx, delegations); err != nil {
		return err
	}

	return c.emitDelegatorUnbondsEvent(ctx, delegator.MarshalPB(), req.Amount)
}

func (c *DPOS) CheckDelegation(ctx contract.StaticContext, req *CheckDelegationRequest) (*CheckDelegationResponse, error) {
	ctx.Logger().Debug("DPOS CheckDelegation", "request", req)

	if req.ValidatorAddress == nil {
		return nil, logStaticDposError(ctx, errors.New("CheckDelegation called with req.ValidatorAddress == nil"), req.String())
	}
	if req.DelegatorAddress == nil {
		return nil, logStaticDposError(ctx, errors.New("CheckDelegation called with req.DelegatorAddress == nil"), req.String())
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	delegation := delegations.Get(*req.ValidatorAddress, *req.DelegatorAddress)
	if delegation == nil {
		return &CheckDelegationResponse{Delegation: &Delegation{
			Validator: req.ValidatorAddress,
			Delegator: req.DelegatorAddress,
			Amount:    loom.BigZeroPB(),
		}}, nil
	} else {
		return &CheckDelegationResponse{Delegation: delegation}, nil
	}
}

func (c *DPOS) TotalDelegation(ctx contract.StaticContext, req *TotalDelegationRequest) (*TotalDelegationResponse, error) {
	ctx.Logger().Debug("DPOS TotalDelegation", "request", req)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

	if req.DelegatorAddress == nil {
		return nil, logStaticDposError(ctx, errors.New("TotalDelegation called with req.DelegatorAddress == nil"), req.String())
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	totalDelegationAmount := common.BigZero()
	totalWeightedDelegationAmount := common.BigZero()
	for _, delegation := range delegations {
		if delegation.Delegator.Local.Compare(req.DelegatorAddress.Local) == 0 {
			totalDelegationAmount.Add(totalDelegationAmount, &delegation.Amount.Value)
			weightedAmount := calculateWeightedDelegationAmount(*delegation, v2_1)
			totalWeightedDelegationAmount.Add(totalWeightedDelegationAmount, &weightedAmount)
		}
	}

	return &TotalDelegationResponse{Amount: &types.BigUInt{Value: *totalDelegationAmount}, WeightedAmount: &types.BigUInt{Value: *totalWeightedDelegationAmount}}, nil
}

func (c *DPOS) CheckAllDelegations(ctx contract.StaticContext, req *CheckAllDelegationsRequest) (*CheckAllDelegationsResponse, error) {
	ctx.Logger().Debug("DPOS CheckAllDelegations", "request", req)

	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)
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
	for _, delegation := range delegations {
		if delegation.Delegator.Local.Compare(req.DelegatorAddress.Local) == 0 {
			totalDelegationAmount.Add(totalDelegationAmount, &delegation.Amount.Value)
			weightedAmount := calculateWeightedDelegationAmount(*delegation, v2_1)
			totalWeightedDelegationAmount.Add(totalWeightedDelegationAmount, &weightedAmount)
			delegatorDelegations = append(delegatorDelegations, delegation)
		}
	}

	return &CheckAllDelegationsResponse{Amount: &types.BigUInt{Value: *totalDelegationAmount}, WeightedAmount: &types.BigUInt{Value: *totalWeightedDelegationAmount}, Delegations: delegatorDelegations}, nil
}

// **************************
// CANDIDATE REGISTRATION
// **************************

func (c *DPOS) WhitelistCandidate(ctx contract.Context, req *WhitelistCandidateRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
			WhitelistLocktime: req.LockTime, // REPURPOSED: Now is used in the same way as a LockTimeTier and valid values can be 0-3. Any other value will be considered to give normal 5% bonuses.
			DistributionTotal: loom.BigZeroPB(),
			DelegationTotal:   loom.BigZeroPB(),
			SlashPercentage:   loom.BigZeroPB(),
		})
	} else {
		// ValidatorStatistic must not yet exist for a particular candidate in order
		// to be whitelisted
		return logDposError(ctx, errors.New("Cannot whitelist an already whitelisted candidate."), req.String())
	}

	return saveValidatorStatisticList(ctx, statistics)
}

func (c *DPOS) RemoveWhitelistedCandidate(ctx contract.Context, req *RemoveWhitelistedCandidateRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	statistic := statistics.Get(loom.UnmarshalAddressPB(req.CandidateAddress))

	if statistic == nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	} else {
		statistic.WhitelistLocktime = 0
		statistic.WhitelistAmount = loom.BigZeroPB()
	}

	return saveValidatorStatisticList(ctx, statistics)
}

func (c *DPOS) ChangeWhitelistAmount(ctx contract.Context, req *ChangeWhitelistAmountRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	statistic := statistics.Get(loom.UnmarshalAddressPB(req.CandidateAddress))
	if statistic == nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	} else {
		statistic.WhitelistAmount = req.Amount
	}
	return saveValidatorStatisticList(ctx, statistics)
}

func (c *DPOS) ChangeWhitelistLockTimeTier(ctx contract.Context, req *ChangeWhitelistLockTimeTierRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOS", "ChangeWhitelistLockTimeTier", "sender", sender, "request", req)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())

	}
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	statistic := statistics.Get(loom.UnmarshalAddressPB(req.CandidateAddress))
	if statistic == nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	} else {
		statistic.WhitelistLocktime = req.LockTimeTier

	}
	return saveValidatorStatisticList(ctx, statistics)
}

func (c *DPOS) RegisterCandidate2(ctx contract.Context, req *RegisterCandidateRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	candidateAddress := ctx.Message().Sender
	ctx.Logger().Info("DPOS RegisterCandidate", "candidate", candidateAddress, "request", req)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

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
		return logDposError(ctx, errCandidateNotFound, req.String())
	}

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	statistic := statistics.Get(candidateAddress)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if (statistic == nil || common.IsZero(statistic.WhitelistAmount.Value)) && common.IsPositive(state.Params.RegistrationRequirement.Value) {
		// A currently unregistered candidate must make a loom token deposit
		// = 'registrationRequirement' in order to run for validator.
		coin := loadCoin(ctx, state.Params)

		dposContractAddress := ctx.ContractAddress()
		err = coin.TransferFrom(candidateAddress, dposContractAddress, &state.Params.RegistrationRequirement.Value)
		if err != nil {
			return err
		}

		delegations, err := loadDelegationList(ctx)
		if err != nil {
			return err
		}

		// Self-delegate funds for the amount of time specified
		tier := req.GetLocktimeTier()
		if tier > 3 {
			return logDposError(ctx, errors.New("Invalid locktime tier"), req.String())
		}

		locktimeTier := TierMap[tier]
		now := uint64(ctx.Now().Unix())
		var tierTime uint64
		if !v2_1 {
			tierTime = calculateTierLocktime(locktimeTier, uint64(state.Params.ElectionCycleLength))
		} else {
			tierTime = TierLocktimeMap[locktimeTier]
		}
		lockTime := now + tierTime

		delegation := &Delegation{
			Validator:    candidateAddress.MarshalPB(),
			Delegator:    candidateAddress.MarshalPB(),
			Amount:       loom.BigZeroPB(),
			UpdateAmount: state.Params.RegistrationRequirement,
			Height:       uint64(ctx.Block().Height),
			LocktimeTier: locktimeTier,
			LockTime:     lockTime,
			State:        BONDING,
		}
		delegations.Set(delegation)

		if err = saveDelegationList(ctx, delegations); err != nil {
			return err
		}
	}

	newCandidate := &dtypes.CandidateV2{
		PubKey:      req.PubKey,
		Address:     candidateAddress.MarshalPB(),
		Fee:         req.Fee,
		NewFee:      req.Fee,
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
	}
	candidates.Set(newCandidate)

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateRegistersEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) RegisterCandidate(ctx contract.Context, req *RegisterCandidateRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	candidateAddress := ctx.Message().Sender
	ctx.Logger().Info("DPOS RegisterCandidate", "candidate", candidateAddress, "request", req)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

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
		return logDposError(ctx, errCandidateNotFound, req.String())
	}

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	statistic := statistics.Get(candidateAddress)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if (statistic == nil || common.IsZero(statistic.WhitelistAmount.Value)) && common.IsPositive(state.Params.RegistrationRequirement.Value) {
		// A currently unregistered candidate must make a loom token deposit
		// = 'registrationRequirement' in order to run for validator.
		coin := loadCoin(ctx, state.Params)

		dposContractAddress := ctx.ContractAddress()
		err = coin.TransferFrom(candidateAddress, dposContractAddress, &state.Params.RegistrationRequirement.Value)
		if err != nil {
			return err
		}

		delegations, err := loadDelegationList(ctx)
		if err != nil {
			return err
		}

		// Self-delegate funds for the amount of time specified
		tier := req.GetLocktimeTier()
		if tier > 3 {
			return logDposError(ctx, errors.New("Invalid locktime tier"), req.String())
		}

		locktimeTier := TierMap[tier]
		now := uint64(ctx.Now().Unix())
		var tierTime uint64
		if !v2_1 {
			tierTime = calculateTierLocktime(locktimeTier, uint64(state.Params.ElectionCycleLength))
		} else {
			tierTime = TierLocktimeMap[locktimeTier]
		}
		lockTime := now + tierTime

		delegation := &Delegation{
			Validator:    candidateAddress.MarshalPB(),
			Delegator:    candidateAddress.MarshalPB(),
			Amount:       loom.BigZeroPB(),
			UpdateAmount: state.Params.RegistrationRequirement,
			Height:       uint64(ctx.Block().Height),
			LocktimeTier: locktimeTier,
			LockTime:     lockTime,
			State:        BONDING,
		}
		delegations.Set(delegation)

		if err = saveDelegationList(ctx, delegations); err != nil {
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

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateRegistersEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) ChangeFee(ctx contract.Context, req *ChangeCandidateFeeRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	cand.NewFee = req.Fee
	cand.FeeDelayCounter = 0

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateFeeChangeEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) UpdateCandidateInfo(ctx contract.Context, req *UpdateCandidateInfoRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

	candidateAddress := ctx.Message().Sender
	ctx.Logger().Info("DPOS RemoveWhitelistCandidate", "candidateAddress", candidateAddress, "request", req)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return logDposError(ctx, errCandidateNotFound, req.String())
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

		slashValidatorDelegations(&delegations, statistic, candidateAddress, v2_1)

		// reset validator self-delegation
		delegation := delegations.Get(*candidateAddress.MarshalPB(), *candidateAddress.MarshalPB())

		// In case that a whitelisted candidate with no self-delegation calls this
		// function, we must check that delegation is not nil
		if delegation != nil {
			if delegation.LockTime > uint64(ctx.Now().Unix()) {
				return logDposError(ctx, errors.New("Validator's self-delegation currently locked."), req.String())
			} else if delegation.State != BONDED {
				return logDposError(ctx, errors.New("Existing delegation not in BONDED state."), req.String())
			} else {
				// Once this delegation is unbonded, the total self-delegation
				// amount will be returned to the unregistered validator
				delegation.State = UNBONDING
				delegation.UpdateAmount = &types.BigUInt{Value: delegation.Amount.Value}
				delegations.Set(delegation)
				if err = saveDelegationList(ctx, delegations); err != nil {
					return err
				}
			}
		}
	}

	// Remove canidate from candidates array
	candidates.Delete(candidateAddress)
	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateUnregistersEvent(ctx, candidateAddress.MarshalPB())
}

func (c *DPOS) ListCandidates(ctx contract.StaticContext, req *ListCandidateRequest) (*ListCandidateResponse, error) {
	ctx.Logger().Debug("DPOS ListCandidates", "request", req)

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
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
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// Check if enough time has elapsed to start new validator election
	if state.Params.ElectionCycleLength > (ctx.Now().Unix() - state.LastElectionTime) {
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

	// Update each candidate's fee
	for _, c := range candidates {
		if c.Fee != c.NewFee {
			c.FeeDelayCounter += 1
			if c.FeeDelayCounter == feeChangeDelay {
				c.Fee = c.NewFee
			}
		}
	}
	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	// When there are no token delegations and no statistics (which contain
	// whitelist delegation amounts), quit the function early and leave the
	// validators as they are
	if len(delegations) == 0 && len(statistics) == 0 {
		return nil
	}

	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return err
	}

	formerValidatorTotals, delegatorRewards := rewardAndSlash(state, candidates, &statistics, &delegations, &distributions, v2_1)
	newDelegationTotals, err := distributeDelegatorRewards(ctx, *state, formerValidatorTotals, delegatorRewards, &delegations, &distributions, &statistics)
	if err != nil {
		return err
	}
	// save delegation updates that occurred in distributeDelegatorRewards
	if err = saveDelegationList(ctx, delegations); err != nil {
		return err
	}

	if err = saveDistributionList(ctx, distributions); err != nil {
		return err
	}

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
	totalValidatorDelegations := common.BigZero()
	for _, res := range delegationResults[:validatorCount] {
		candidate := candidates.Get(res.ValidatorAddress)
		if candidate != nil {
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

	if err = saveValidatorStatisticList(ctx, statistics); err != nil {
		return err
	}

	state.Validators = validators
	state.LastElectionTime = ctx.Now().Unix()
	state.TotalValidatorDelegations = &types.BigUInt{Value: *totalValidatorDelegations}

	ctx.Logger().Debug("DPOS Elect", "Post-Elect State", state)
	if err = saveState(ctx, state); err != nil {
		return err
	}

	return emitElectionEvent(ctx)
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

	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	chainID := ctx.Block().ChainID

	displayStatistics := make([]*ValidatorStatistic, 0)
	for _, validator := range validators {
		address := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(validator.PubKey)}

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

// ListValidatorsSimple returns the current validator set without statistics.
func (c *DPOS) ListValidatorsSimple(
	ctx contract.StaticContext, req *ListValidatorsRequest,
) (*ListValidatorsResponse, error) {
	validators, err := ValidatorList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	displayStatistics := make([]*ValidatorStatistic, 0, len(validators))

	for _, validator := range validators {
		address := loom.Address{
			ChainID: ctx.Block().ChainID,
			Local:   loom.LocalAddressFromPublicKey(validator.PubKey),
		}
		stat := &ValidatorStatistic{
			PubKey:  validator.PubKey,
			Address: address.MarshalPB(),
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

	if req.Candidate == nil {
		return nil, logStaticDposError(ctx, errors.New("ListDelegations called with req.Candidate == nil"), req.String())
	}

	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	total := common.BigZero()
	candidateDelegations := make([]*Delegation, 0)
	for _, delegation := range delegations {
		if delegation.Validator.Local.Compare(req.Candidate.Local) == 0 {
			candidateDelegations = append(candidateDelegations, delegation)
			total = total.Add(total, &delegation.Amount.Value)
		}
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
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return err
	}
	stat := statistics.GetV2(validatorAddr)
	if stat == nil {
		return logDposError(ctx, errors.New("Cannot slash default validator."), "")
	}

	// If slashing percentage is less than current total slash percentage, do
	// not further increase total slash percentage during this election period
	if slashPercentage.Cmp(&stat.SlashPercentage.Value) < 0 {
		return nil
	}

	updatedAmount := common.BigZero()
	updatedAmount.Add(&stat.SlashPercentage.Value, &slashPercentage)
	stat.SlashPercentage = &types.BigUInt{Value: *updatedAmount}

	if err = saveValidatorStatisticList(ctx, statistics); err != nil {
		return err
	}

	return emitSlashEvent(ctx, stat.Address, slashPercentage)
}

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

func loadCoin(ctx contract.Context, params *Params) *ERC20 {
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)
	return &ERC20{
		Context:         ctx,
		ContractAddress: coinAddr,
	}
}

func loadStaticCoin(ctx contract.StaticContext, params *Params) *ERC20Static {
	coinAddr := loom.UnmarshalAddressPB(params.CoinContractAddress)
	return &ERC20Static{
		StaticContext:   ctx,
		ContractAddress: coinAddr,
	}
}

// rewards & slashes are calculated along with former delegation totals
// rewards are distributed to validators based on fee
// rewards distribution amounts are prepared for delegators
func rewardAndSlash(state *State, candidates CandidateList, statistics *ValidatorStatisticList, delegations *DelegationList, distributions *DistributionList, v2_1 bool) (map[string]loom.BigUInt, map[string]*loom.BigUInt) {
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
					rewardValidator(statistic, state.Params, state.TotalValidatorDelegations.Value, v2_1)
					validatorShare := CalculateFraction(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, statistic.DistributionTotal.Value, v2_1)

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
						amount := statistic.WhitelistAmount.Value
						if v2_1 {
							amount = calculateWeightedWhitelistAmount(*statistic)
						}
						whitelistDistribution := calculateShare(amount, statistic.DelegationTotal.Value, *delegatorsShare, v2_1)
						// increase a delegator's distribution
						distributions.IncreaseDistribution(*candidate.Address, whitelistDistribution)
					}
				} else {
					slashValidatorDelegations(delegations, statistic, candidateAddress, v2_1)
				}

				// Zeroing out validator's distribution total since it will be transferred
				// to the distributions storage during this `Elect` call.
				// Validators and Delegators both can claim their rewards in the
				// same way when this is true.
				state.TotalRewardDistribution.Value.Add(&state.TotalRewardDistribution.Value, &statistic.DistributionTotal.Value)
				statistic.DistributionTotal = loom.BigZeroPB()
				formerValidatorTotals[validatorKey] = statistic.DelegationTotal.Value
			}
		}
	}
	return formerValidatorTotals, delegatorRewards
}

func rewardValidator(statistic *ValidatorStatistic, params *Params, totalValidatorDelegations loom.BigUInt, v2_1 bool) {
	// if there is no slashing to be applied, reward validator
	cycleSeconds := params.ElectionCycleLength
	reward := CalculateFraction(blockRewardPercentage, statistic.DelegationTotal.Value, v2_1)

	// if totalValidator Delegations are high enough to make simple reward
	// calculations result in more rewards given out than the value of `MaxYearlyReward`,
	// scale the rewards appropriately
	yearlyRewardTotal := CalculateFraction(blockRewardPercentage, totalValidatorDelegations, v2_1)
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

func slashValidatorDelegations(delegations *DelegationList, statistic *ValidatorStatistic, validatorAddress loom.Address, v2_1 bool) {
	// these delegation totals will be added back up again when we calculate new delegation totals below
	for _, delegation := range *delegations {
		// check the it's a delegation that belongs to the validator
		if delegation.Validator.Local.Compare(validatorAddress.Local) == 0 {
			toSlash := CalculateFraction(statistic.SlashPercentage.Value, delegation.Amount.Value, v2_1)
			updatedAmount := common.BigZero()
			updatedAmount.Sub(&delegation.Amount.Value, &toSlash)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
		}
	}

	// Slash a whitelisted candidate's whitelist amount. This doesn't affect how
	// much the validator gets back from token timelock, but will decrease the
	// validator's delegation total & thus his ability to earn rewards
	if !common.IsZero(statistic.WhitelistAmount.Value) {
		toSlash := CalculateFraction(statistic.SlashPercentage.Value, statistic.WhitelistAmount.Value, v2_1)
		updatedAmount := common.BigZero()
		updatedAmount.Sub(&statistic.WhitelistAmount.Value, &toSlash)
		statistic.WhitelistAmount = &types.BigUInt{Value: *updatedAmount}
	}

	// reset slash total
	statistic.SlashPercentage = loom.BigZeroPB()
}

// This function has three goals 1) distribute a validator's rewards to each of
// the delegators, 2) finalize the bonding process for any delegations received
// during the last election period (delegate & unbond calls) and 3) calculate
// the new delegation totals.
func distributeDelegatorRewards(ctx contract.Context, state State, formerValidatorTotals map[string]loom.BigUInt, delegatorRewards map[string]*loom.BigUInt, delegations *DelegationList, distributions *DistributionList, statistics *ValidatorStatisticList) (map[string]*loom.BigUInt, error) {
	newDelegationTotals := make(map[string]*loom.BigUInt)
	v2_1 := ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false)

	// initialize delegation totals with whitelist amounts
	for _, statistic := range *statistics {
		if statistic.WhitelistAmount != nil && !common.IsZero(statistic.WhitelistAmount.Value) {
			amount := statistic.WhitelistAmount.Value
			if ctx.FeatureEnabled(loomchain.DPOSVersion2_1, false) {
				amount = calculateWeightedWhitelistAmount(*statistic)
			}
			validatorKey := loom.UnmarshalAddressPB(statistic.Address).String()
			newDelegationTotals[validatorKey] = &amount
		}
	}

	for _, delegation := range *delegations {
		validatorKey := loom.UnmarshalAddressPB(delegation.Validator).String()

		// Do do distribute rewards to delegators of the Limbo validators
		if delegation.Validator.Local.Compare(limboValidatorAddress.Local) != 0 {
			// allocating validator distributions to delegators
			// based on former validator delegation totals
			delegationTotal := formerValidatorTotals[validatorKey]
			rewardsTotal := delegatorRewards[validatorKey]
			if rewardsTotal != nil {
				weightedDelegation := calculateWeightedDelegationAmount(*delegation, v2_1)
				delegatorDistribution := calculateShare(weightedDelegation, delegationTotal, *rewardsTotal, v2_1)
				// increase a delegator's distribution
				distributions.IncreaseDistribution(*delegation.Delegator, delegatorDistribution)
			}
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
		} else if delegation.State == REDELEGATING {
			delegation.Validator = delegation.UpdateValidator
			validatorKey = loom.UnmarshalAddressPB(delegation.Validator).String()
		}

		// After a delegation update, zero out UpdateAmount
		delegation.UpdateAmount = loom.BigZeroPB()
		delegation.State = BONDED

		// Do do calculate delegation total of the Limbo validators
		if delegation.Validator.Local.Compare(limboValidatorAddress.Local) != 0 {
			newTotal := common.BigZero()
			weightedDelegation := calculateWeightedDelegationAmount(*delegation, v2_1)
			newTotal.Add(newTotal, &weightedDelegation)
			if newDelegationTotals[validatorKey] != nil {
				newTotal.Add(newTotal, newDelegationTotals[validatorKey])
			}
			newDelegationTotals[validatorKey] = newTotal
		}
	}

	return newDelegationTotals, nil
}

func (c *DPOS) ClaimDistribution(ctx contract.Context, req *ClaimDistributionRequest) (*ClaimDistributionResponse, error) {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return nil, logDposError(ctx, errContractDisabled, req.String())
	}

	ctx.Logger().Info("DPOS ClaimDistribution", "request", req)

	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return nil, err
	}

	delegator := ctx.Message().Sender

	distribution := distributions.Get(*delegator.MarshalPB())
	if distribution == nil {
		return nil, logDposError(ctx, errors.New(fmt.Sprintf("distribution not found: %s", delegator)), req.String())
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

	resp := &ClaimDistributionResponse{Amount: &types.BigUInt{Value: distribution.Amount.Value}}

	err = distributions.ResetTotal(*delegator.MarshalPB())
	if err != nil {
		return nil, err
	}

	ctx.Logger().Info("DPOS ClaimDistribution result", "delegator", delegator, "amount", distribution.Amount)

	err = c.emitDelegatorClaimsRewardsEvent(ctx, delegator.MarshalPB(), distribution.Amount)
	if err != nil {
		return nil, err
	}

	err = saveDistributionList(ctx, distributions)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *DPOS) CheckDistribution(ctx contract.StaticContext, req *CheckDistributionRequest) (*CheckDistributionResponse, error) {
	if req.Address == nil {
		return nil, logStaticDposError(ctx, errors.New("Must provide a valid address"), req.String())
	}

	ctx.Logger().Debug("DPOS CheckDistribution", "delegator", *req.Address, "request", req)

	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return nil, err
	}

	distribution := distributions.Get(*req.Address)
	if distribution == nil {
		return nil, logStaticDposError(ctx, errDistributionNotFound, req.String())
	}
	resp := &CheckDistributionResponse{
		Amount: distribution.Amount,
	}

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

func (c *DPOS) GetDistributions(ctx contract.StaticContext, req *GetDistributionsRequest) (*GetDistributionsResponse, error) {
	ctx.Logger().Debug("DPOS", "GetDistributions", "request", req)

	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	return &GetDistributionsResponse{
		Distributions: distributions,
	}, nil

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
		case *dtypes.BatchRequestV2_WhitelistCandidate:
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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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

func (c *DPOS) SetValidatorCount2(ctx contract.Context, req *SetValidatorCountRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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

	state.Params.ValidatorCount = uint64(req.ValidatorCount)

	return saveState(ctx, state)
}

func (c *DPOS) SetValidatorCount(ctx contract.Context, req *SetValidatorCountRequest) error {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return logDposError(ctx, errContractDisabled, req.String())
	}

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

func (c *DPOS) emitDelegatorClaimsRewardsEvent(ctx contract.Context, delegator *types.Address, amount *types.BigUInt) error {
	marshalled, err := proto.Marshal(&DposDelegatorClaimsRewardsEvent{
		Address: delegator,
		Amount:  amount,
		Time:    ctx.Block().Time,
		Height:  ctx.Block().Height,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorClaimsRewardsEventTopic)
	return nil
}

// ***************************
// MIGRATION FUNCTIONS
// ***************************

func Dump(ctx contract.Context, dposv3Address loom.Address) (*dposv3.InitializationState, error) {
	if ctx.FeatureEnabled(feature.DPOSVersion3Feature, false) {
		return nil, logDposError(ctx, errContractDisabled, "DPOSv2 Dump called")
	}

	ctx.Logger().Info("DPOSv2 Dump")
	sender := ctx.Message().Sender

	// load v2 state and pack it into v3 state
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Local.Compare(state.Params.OracleAddress.Local) != 0 {
		return nil, logDposError(ctx, errOnlyOracle, "DPOSv2 Dump")
	}

	initializationState, err := populateInitializationState(ctx, state)
	if err != nil {
		return nil, err
	}

	staticCoin := loadStaticCoin(ctx, state.Params)
	dposv2Addr := ctx.ContractAddress()
	// send all dposv2 funds to dposv3 (representing unpaid rewards & delegations)
	balanceResponse, err := staticCoin.BalanceOf(dposv2Addr)
	if err != nil {
		return nil, err
	}

	coin := loadCoin(ctx, state.Params)
	err = coin.Transfer(dposv3Address, balanceResponse)
	if err != nil {
		return nil, err
	}

	return initializationState, nil
}

func (c *DPOS) ViewStateDump(ctx contract.StaticContext, req *ViewStateDumpRequest) (*ViewStateDumpResponse, error) {
	ctx.Logger().Debug("DPOS ViewStateDump", "request", req)

	// load v2 state and pack it into v3 state
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, err
	}
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}
	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return nil, err
	}
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return nil, err
	}

	currentV2State := &StateDump{
		State:         state,
		Candidates:    candidates,
		Delegations:   delegations,
		Distributions: distributions,
		Statistics:    statistics,
	}

	initializationState, err := populateInitializationState(ctx, state)
	if err != nil {
		return nil, err
	}

	// Checking that number of Candidates does not change
	if len(candidates) != len(initializationState.Candidates) {
		return nil, logStaticDposError(ctx, errors.New("Migration resulted in unequal number of Candidates."), req.String())
	}

	// Checking that number of Delegations/Distributions does not change
	if (len(delegations) + len(distributions)) != len(initializationState.Delegations) {
		return nil, logStaticDposError(ctx, errors.New("Migration resulted in len(v2Delegations + v2Distributions) != len(v3Delegations)."), req.String())
	}

	// Checking that number of statistics does not change
	if len(statistics) != len(initializationState.Statistics) {
		return nil, logStaticDposError(ctx, errors.New("Migration resulted in unequal number of ValidatorStatistics."), req.String())
	}

	// Checking that whitelist totals & weighted totals do not change
	// Note: checking the total amount is enough
	whitelistTotalV2 := common.BigZero()
	for _, statistic := range statistics {
		whitelistTotalV2.Add(whitelistTotalV2, &statistic.WhitelistAmount.Value)
	}

	whitelistTotalV3 := common.BigZero()
	for _, statistic := range initializationState.Statistics {
		whitelistTotalV3.Add(whitelistTotalV3, &statistic.WhitelistAmount.Value)
	}

	if whitelistTotalV2.Cmp(whitelistTotalV3) != 0 {
		return nil, logStaticDposError(ctx, errors.New("Migration resulted in inconsistent whitelist amounts."), req.String())
	}

	// Checking that (non-zero index) delegation sum does not change
	delegationTotalV2 := common.BigZero()
	for _, delegation := range delegations {
		delegationTotalV2.Add(delegationTotalV2, &adjustDoubledDelegationAmount(*delegation).Value)
	}

	delegationTotalV3 := common.BigZero()
	distributionTotalV3 := common.BigZero()
	for _, delegation := range initializationState.Delegations {
		if delegation.Index != 0 {
			delegationTotalV3.Add(delegationTotalV3, &delegation.Amount.Value)
		} else {
			distributionTotalV3.Add(distributionTotalV3, &delegation.Amount.Value)
		}
	}

	if delegationTotalV2.Cmp(delegationTotalV3) != 0 {
		return nil, logStaticDposError(ctx, errors.New("Migration resulted in inconsistent delegation amounts."), req.String())
	}

	// Checking that distributions / zero-index delegations sum does not change
	distributionTotalV2 := common.BigZero()
	for _, distribution := range distributions {
		distributionTotalV2.Add(distributionTotalV2, &distribution.Amount.Value)
	}

	if distributionTotalV2.Cmp(distributionTotalV3) != 0 {
		return nil, logStaticDposError(ctx, errors.New("Migration resulted in inconsistent distribution amounts."), req.String())
	}

	resp := &ViewStateDumpResponse{
		OldState: currentV2State,
		NewState: initializationState,
	}

	return resp, nil
}

func populateInitializationState(ctx contract.StaticContext, state *State) (*dposv3.InitializationState, error) {
	v3Params := &dposv3.Params{
		ValidatorCount:              state.Params.ValidatorCount,
		ElectionCycleLength:         state.Params.ElectionCycleLength,
		CoinContractAddress:         state.Params.CoinContractAddress,
		OracleAddress:               state.Params.OracleAddress,
		MaxYearlyReward:             state.Params.MaxYearlyReward,
		RegistrationRequirement:     state.Params.RegistrationRequirement,
		CrashSlashingPercentage:     state.Params.CrashSlashingPercentage,
		ByzantineSlashingPercentage: state.Params.ByzantineSlashingPercentage,
	}

	v3State := &dposv3.State{
		Params:                    v3Params,
		Validators:                state.Validators,
		LastElectionTime:          state.LastElectionTime,
		TotalValidatorDelegations: state.TotalValidatorDelegations,
		TotalRewardDistribution:   state.TotalRewardDistribution,
	}

	// load v2 Candidates and pack them into v3 Candidates
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	var v3Candidates []*dposv3.Candidate
	for _, candidate := range candidates {
		v3Candidate := &dposv3.Candidate{
			Address: candidate.Address,
			PubKey:  candidate.PubKey,
			Fee:     candidate.Fee,
			NewFee:  candidate.NewFee,
			// Any candidate mid-fee change during migration will have to call
			// ChangeFee again
			State:       dposv3.REGISTERED,
			Name:        candidate.Name,
			Description: candidate.Description,
			Website:     candidate.Website,
		}
		v3Candidates = append(v3Candidates, v3Candidate)
	}

	// load v2 Statistics and pack them into v3 Statistics
	statistics, err := loadValidatorStatisticList(ctx)
	if err != nil {
		return nil, err
	}

	var v3Statistics []*dposv3.ValidatorStatistic
	for _, statistic := range statistics {
		v3Statistic := &dposv3.ValidatorStatistic{
			Address:         statistic.Address,
			WhitelistAmount: statistic.WhitelistAmount,
			LocktimeTier:    dposv3.TierMap[statistic.WhitelistLocktime],
			DelegationTotal: statistic.DelegationTotal,
			SlashPercentage: statistic.SlashPercentage,
		}
		v3Statistics = append(v3Statistics, v3Statistic)
	}

	var v3Delegations []*dposv3.Delegation
	// load v2 Distributions and pack them into v3 Delegations @ index 0
	distributions, err := loadDistributionList(ctx)
	if err != nil {
		return nil, err
	}

	for _, distribution := range distributions {
		v3Delegation := &dposv3.Delegation{
			Validator:    dposv3.LimboValidatorAddress(ctx).MarshalPB(),
			Delegator:    distribution.Address,
			Index:        dposv3.REWARD_DELEGATION_INDEX,
			Amount:       distribution.Amount,
			UpdateAmount: loom.BigZeroPB(),
			LockTime:     0,
			LocktimeTier: dposv3.TIER_ZERO,
			State:        dposv3.BONDED,
		}
		v3Delegations = append(v3Delegations, v3Delegation)
	}

	// load v2 Delegations and pack them into v3 Delegations @ index 1
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	// in case of duplicates create an array to store delegation index
	indices := make(map[string]uint64)
	for _, delegation := range delegations {
		validatorString := delegation.Validator.Local.String()
		delegatorString := delegation.Delegator.Local.String()
		delegationKey := validatorString + delegatorString
		validator := adjustValidatorIfInPlasmaValidators(*delegation)
		amount := adjustDoubledDelegationAmount(*delegation)

		v3Delegation := &dposv3.Delegation{
			Validator:    adjustValidatorIfLimboValidator(ctx, validator),
			Delegator:    delegation.Delegator,
			Index:        dposv3.DELEGATION_START_INDEX + indices[delegationKey],
			Amount:       amount,
			UpdateAmount: delegation.UpdateAmount,
			LockTime:     delegation.LockTime,
			LocktimeTier: dposv3.TierMap[uint64(delegation.LocktimeTier)],
			// All delegations are BONDED when migrated. Otherwise, it'd be
			// difficult to test consistency across a migration.
			State: dposv3.BONDED,
		}
		v3Delegations = append(v3Delegations, v3Delegation)

		indices[delegationKey]++
	}

	initializationState := &dposv3.InitializationState{
		State:       v3State,
		Candidates:  v3Candidates,
		Statistics:  v3Statistics,
		Delegations: v3Delegations,
	}

	return initializationState, nil
}
