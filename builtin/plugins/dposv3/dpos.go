package dposv3

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/features"
	"github.com/pkg/errors"
)

const (
	defaultDowntimePeriod          = 4096
	defaultRegistrationRequirement = 1250000
	defaultMaxYearlyReward         = 60000000
	tokenDecimals                  = 18
	billionthsBasisPointRatio      = 100000
	hundredPercentInBasisPoints    = 10000
	yearSeconds                    = int64(60 * 60 * 24 * 365)
	BONDING                        = dtypes.Delegation_BONDING
	BONDED                         = dtypes.Delegation_BONDED
	UNBONDING                      = dtypes.Delegation_UNBONDING
	REDELEGATING                   = dtypes.Delegation_REDELEGATING
	REGISTERED                     = dtypes.Candidate_REGISTERED
	UNREGISTERING                  = dtypes.Candidate_UNREGISTERING
	ABOUT_TO_CHANGE_FEE            = dtypes.Candidate_ABOUT_TO_CHANGE_FEE
	CHANGING_FEE                   = dtypes.Candidate_CHANGING_FEE
	TIER_ZERO                      = dtypes.LocktimeTier_TIER_ZERO
	TIER_ONE                       = dtypes.LocktimeTier_TIER_ONE
	TIER_TWO                       = dtypes.LocktimeTier_TIER_TWO
	TIER_THREE                     = dtypes.LocktimeTier_TIER_THREE

	ElectionEventTopic               = "dposv3:election"
	SlashEventTopic                  = "dposv3:slash"
	SlashDelegationEventTopic        = "dposv3:slashdelegation"
	SlashWhitelistAmountEventTopic   = "dposv3:slashwhitelistamount"
	JailEventTopic                   = "dposv3:jail"
	UnjailEventTopic                 = "dposv3:unjail"
	CandidateRegistersEventTopic     = "dposv3:candidateregisters"
	CandidateUnregistersEventTopic   = "dposv3:candidateunregisters"
	CandidateFeeChangeEventTopic     = "dposv3:candidatefeechange"
	UpdateCandidateInfoEventTopic    = "dposv3:updatecandidateinfo"
	DelegatorDelegatesEventTopic     = "dposv3:delegatordelegates"
	DelegatorRedelegatesEventTopic   = "dposv3:delegatorredelegates"
	DelegatorConsolidatesEventTopic  = "dposv3:delegatorconsolidates"
	DelegatorUnbondsEventTopic       = "dposv3:delegatorunbonds"
	ReferrerRegistersEventTopic      = "dposv3:referrerregisters"
	DelegatorClaimsRewardsEventTopic = "dposv3:delegatorclaimsrewards"
)

var (
	secondsInYear                    = loom.BigUInt{big.NewInt(yearSeconds)}
	billionth                        = loom.BigUInt{big.NewInt(1000000000)}
	defaultFee                       = uint64(2500) // 25%
	defaultReferrerFee               = loom.BigUInt{big.NewInt(300)}
	blockRewardPercentage            = loom.BigUInt{big.NewInt(500)}
	doubleSignSlashPercentage        = loom.BigUInt{big.NewInt(500)}
	defaultInactivitySlashPercentage = loom.BigUInt{big.NewInt(100)}
	defaultMaxDowntimePercentage     = loom.BigUInt{big.NewInt(5000)}
	powerCorrection                  = big.NewInt(1000000000000)
	errCandidateNotFound             = errors.New("Candidate record not found.")
	errStatisticNotFound             = errors.New("Candidate statistic not found.")
	errCandidateAlreadyRegistered    = errors.New("Candidate already registered.")
	errCandidateUnregistering        = errors.New("Candidate is currently unregistering.")
	errValidatorNotFound             = errors.New("Validator record not found.")
	errDistributionNotFound          = errors.New("Distribution record not found.")
	errOnlyOracle                    = errors.New("Function can only be called with oracle address.")
)

type (
	InitRequest                       = dtypes.DPOSInitRequest
	DelegateRequest                   = dtypes.DelegateRequest
	RedelegateRequest                 = dtypes.RedelegateRequest
	WhitelistCandidateRequest         = dtypes.WhitelistCandidateRequest
	RemoveWhitelistedCandidateRequest = dtypes.RemoveWhitelistedCandidateRequest
	ChangeWhitelistInfoRequest        = dtypes.ChangeWhitelistInfoRequest
	DelegationState                   = dtypes.Delegation_DelegationState
	LocktimeTier                      = dtypes.LocktimeTier
	UnbondRequest                     = dtypes.UnbondRequest
	ConsolidateDelegationsRequest     = dtypes.ConsolidateDelegationsRequest
	CheckAllDelegationsRequest        = dtypes.CheckAllDelegationsRequest
	CheckAllDelegationsResponse       = dtypes.CheckAllDelegationsResponse
	CheckDelegationRequest            = dtypes.CheckDelegationRequest
	CheckDelegationResponse           = dtypes.CheckDelegationResponse
	CheckRewardsRequest               = dtypes.CheckRewardsRequest
	CheckRewardsResponse              = dtypes.CheckRewardsResponse
	CheckRewardDelegationRequest      = dtypes.CheckRewardDelegationRequest
	CheckRewardDelegationResponse     = dtypes.CheckRewardDelegationResponse
	DowntimeRecordRequest             = dtypes.DowntimeRecordRequest
	DowntimeRecordResponse            = dtypes.DowntimeRecordResponse
	DowntimeRecord                    = dtypes.DowntimeRecord
	TimeUntilElectionRequest          = dtypes.TimeUntilElectionRequest
	TimeUntilElectionResponse         = dtypes.TimeUntilElectionResponse
	RegisterCandidateRequest          = dtypes.RegisterCandidateRequest
	ChangeCandidateFeeRequest         = dtypes.ChangeCandidateFeeRequest
	SetMinCandidateFeeRequest         = dtypes.SetMinCandidateFeeRequest
	UpdateCandidateInfoRequest        = dtypes.UpdateCandidateInfoRequest
	UnregisterCandidateRequest        = dtypes.UnregisterCandidateRequest
	ListCandidatesRequest             = dtypes.ListCandidatesRequest
	ListCandidatesResponse            = dtypes.ListCandidatesResponse
	ListValidatorsRequest             = dtypes.ListValidatorsRequest
	ListValidatorsResponse            = dtypes.ListValidatorsResponse
	ListDelegationsRequest            = dtypes.ListDelegationsRequest
	ListDelegationsResponse           = dtypes.ListDelegationsResponse
	ListAllDelegationsRequest         = dtypes.ListAllDelegationsRequest
	ListAllDelegationsResponse        = dtypes.ListAllDelegationsResponse
	Referrer                          = dtypes.Referrer
	ListReferrersRequest              = dtypes.ListReferrersRequest
	ListReferrersResponse             = dtypes.ListReferrersResponse
	RegisterReferrerRequest           = dtypes.RegisterReferrerRequest
	SetDowntimePeriodRequest          = dtypes.SetDowntimePeriodRequest
	SetElectionCycleRequest           = dtypes.SetElectionCycleRequest
	SetMaxYearlyRewardRequest         = dtypes.SetMaxYearlyRewardRequest
	SetRegistrationRequirementRequest = dtypes.SetRegistrationRequirementRequest
	SetValidatorCountRequest          = dtypes.SetValidatorCountRequest
	SetOracleAddressRequest           = dtypes.SetOracleAddressRequest
	SetSlashingPercentagesRequest     = dtypes.SetSlashingPercentagesRequest
	UnjailRequest                     = dtypes.UnjailRequest
	SetMaxDowntimePercentageRequest   = dtypes.SetMaxDowntimePercentageRequest
	EnableValidatorJailingRequest     = dtypes.EnableValidatorJailingRequest
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
	CheckDelegatorRewardsRequest      = dtypes.CheckDelegatorRewardsRequest
	CheckDelegatorRewardsResponse     = dtypes.CheckDelegatorRewardsResponse
	ClaimDelegatorRewardsRequest      = dtypes.ClaimDelegatorRewardsRequest
	ClaimDelegatorRewardsResponse     = dtypes.ClaimDelegatorRewardsResponse

	DposElectionEvent               = dtypes.DposElectionEvent
	DposSlashEvent                  = dtypes.DposSlashEvent
	DposSlashDelegationEvent        = dtypes.DposSlashDelegationEvent
	DposSlashWhitelistAmountEvent   = dtypes.DposSlashWhitelistAmountEvent
	DposJailEvent                   = dtypes.DposJailEvent
	DposUnjailEvent                 = dtypes.DposUnjailEvent
	DposCandidateRegistersEvent     = dtypes.DposCandidateRegistersEvent
	DposCandidateUnregistersEvent   = dtypes.DposCandidateUnregistersEvent
	DposCandidateFeeChangeEvent     = dtypes.DposCandidateFeeChangeEvent
	DposUpdateCandidateInfoEvent    = dtypes.DposUpdateCandidateInfoEvent
	DposDelegatorDelegatesEvent     = dtypes.DposDelegatorDelegatesEvent
	DposDelegatorRedelegatesEvent   = dtypes.DposDelegatorRedelegatesEvent
	DposDelegatorConsolidatesEvent  = dtypes.DposDelegatorConsolidatesEvent
	DposDelegatorUnbondsEvent       = dtypes.DposDelegatorUnbondsEvent
	DposReferrerRegistersEvent      = dtypes.DposReferrerRegistersEvent
	DposDelegatorClaimsRewardsEvent = dtypes.DposDelegatorClaimsRewardsEvent

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
	ctx.Logger().Info("DPOSv3 Init", "Params", req)
	params := req.Params

	if params.CoinContractAddress == nil {
		addr, err := ctx.Resolve("coin")
		if err != nil {
			return err
		}
		params.CoinContractAddress = addr.MarshalPB()
	}
	if params.CrashSlashingPercentage == nil {
		params.CrashSlashingPercentage = &types.BigUInt{Value: defaultInactivitySlashPercentage}
	}
	if params.ByzantineSlashingPercentage == nil {
		params.ByzantineSlashingPercentage = &types.BigUInt{Value: doubleSignSlashPercentage}
	}
	if params.RegistrationRequirement == nil {
		params.RegistrationRequirement = &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}
	}
	if params.MaxDowntimePercentage == nil {
		params.MaxDowntimePercentage = &types.BigUInt{Value: defaultMaxDowntimePercentage}
	}
	if params.MaxYearlyReward == nil {
		params.MaxYearlyReward = &types.BigUInt{Value: *scientificNotation(defaultMaxYearlyReward, tokenDecimals)}
	}
	if params.DowntimePeriod == 0 {
		params.DowntimePeriod = defaultDowntimePeriod
	}

	candidates := &CandidateList{}
	// if InitCandidates is true, whitelist validators and register them for candidates
	if req.InitCandidates {
		for i, validator := range req.Validators {
			candidateAddr := loom.Address{ChainID: ctx.Block().ChainID, Local: loom.LocalAddressFromPublicKey(validator.PubKey)}
			newCandidate := &Candidate{
				PubKey:                validator.PubKey,
				Address:               candidateAddr.MarshalPB(),
				Fee:                   defaultFee,
				NewFee:                defaultFee,
				Name:                  fmt.Sprintf("candidate-%d", i),
				State:                 REGISTERED,
				MaxReferralPercentage: defaultReferrerFee.Uint64(),
			}
			candidates.Set(newCandidate)
			if err := c.addCandidateToStatisticList(ctx, &WhitelistCandidateRequest{
				CandidateAddress: candidateAddr.MarshalPB(),
				Amount:           params.RegistrationRequirement,
				LocktimeTier:     TIER_ZERO,
			}); err != nil {
				return err
			}
		}

		if err := saveCandidateList(ctx, *candidates); err != nil {
			return err
		}
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
	ctx.Logger().Info("DPOSv3 Delegate", "delegator", delegator, "request", req)

	if req.ValidatorAddress == nil {
		return logDposError(ctx, errors.New("Delegate called with req.ValidatorAddress == nil"), req.String())
	}

	cand := GetCandidate(ctx, loom.UnmarshalAddressPB(req.ValidatorAddress))
	// Delegations can only be made to existing candidates
	if cand == nil {
		return logDposError(ctx, errCandidateNotFound, req.String())
	} else if cand.State == UNREGISTERING {
		return logDposError(ctx, errCandidateUnregistering, req.String())
	}

	if req.Amount == nil || !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Must Delegate a positive number of tokens."), req.String())
	}

	// Ensure that referrer value is meaningful
	referrerAddress := getReferrer(ctx, req.Referrer)
	if req.Referrer != "" && referrerAddress == nil {
		return logDposError(ctx, errors.New("Invalid Referrer."), req.String())
	} else if referrerAddress != nil && cand.MaxReferralPercentage < defaultReferrerFee.Uint64() {
		// NOTE: any referral made while a MaxReferralPercentage > ReferrerFee is
		// grandfathered in (i.e. valid) even after a candidate lowers their
		// MaxReferralPercentage
		msg := fmt.Sprintf("Candidate does not accept delegations with referral fees as high. Max: %d, Fee: %d", cand.MaxReferralPercentage, defaultReferrerFee.Uint64())
		return logDposError(ctx, errors.New(msg), req.String())
	}

	coin, err := loadCoin(ctx)
	if err != nil {
		return err
	}

	dposContractAddress := ctx.ContractAddress()
	err = coin.TransferFrom(delegator, dposContractAddress, &req.Amount.Value)
	if err != nil {
		transferFromErr := fmt.Sprintf("Failed coin TransferFrom - Delegate, %v, %s", delegator.String(), req.Amount.Value.String())
		return logDposError(ctx, err, transferFromErr)
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
		Referrer:     req.Referrer, // TODO: This should be a simple index/ID, not a string.
	}

	if err := SetDelegation(ctx, delegation); err != nil {
		return err
	}

	return c.emitDelegatorDelegatesEvent(ctx, delegation)
}

func (c *DPOS) Redelegate(ctx contract.Context, req *RedelegateRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 Redelegate", "delegator", delegator, "request", req)

	if req.ValidatorAddress == nil {
		return logDposError(ctx, errors.New("Redelegate called with req.ValidatorAddress == nil"), req.String())
	}
	if req.FormerValidatorAddress == nil {
		return logDposError(ctx, errors.New("Redelegate called with req.FormerValidatorAddress == nil"), req.String())
	}

	if loom.UnmarshalAddressPB(req.FormerValidatorAddress).Compare(loom.UnmarshalAddressPB(req.ValidatorAddress)) == 0 {
		return logDposError(ctx, errors.New("Redelegating self-delegations is not permitted."), req.String())
	}
	if req.Amount != nil && !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Must Redelegate a positive number of tokens."), req.String())
	}

	// Unless redelegation is to the limbo validator check that the new
	// validator address corresponds to one of the registered candidates
	if loom.UnmarshalAddressPB(req.ValidatorAddress).Compare(LimboValidatorAddress(ctx)) != 0 {
		candidate := GetCandidate(ctx, loom.UnmarshalAddressPB(req.ValidatorAddress))
		// Delegations can only be made to existing candidates
		if candidate == nil {
			return logDposError(ctx, errCandidateNotFound, req.String())
		} else if candidate.State == UNREGISTERING {
			return logDposError(ctx, errCandidateUnregistering, req.String())
		}

		// Ensure that referrer value is meaningful
		referrerAddress := getReferrer(ctx, req.Referrer)
		if req.Referrer != "" && referrerAddress == nil {
			return logDposError(ctx, errors.New("Invalid Referrer."), req.String())
		} else if referrerAddress != nil && candidate.MaxReferralPercentage < defaultReferrerFee.Uint64() {
			msg := fmt.Sprintf("Candidate does not accept delegations with referral fees as high. Max: %d, Fee: %d", candidate.MaxReferralPercentage, defaultReferrerFee.Uint64())
			return logDposError(ctx, errors.New(msg), req.String())
		}
	}

	priorDelegation, err := GetDelegation(ctx, req.Index, *req.FormerValidatorAddress, *delegator.MarshalPB())
	if err == contract.ErrNotFound {
		return logDposError(ctx, errors.New("No delegation to redelegate."), req.String())
	} else if priorDelegation.State != BONDED {
		return logDposError(ctx, errors.New("Cannot redelegate a delegation not in the BONDED state."), req.String())
	} else if err != nil {
		return err
	}

	newLocktimeTier := priorDelegation.LocktimeTier
	newLocktime := priorDelegation.LockTime

	if req.NewLocktimeTier > uint64(newLocktimeTier) {
		state, err := LoadState(ctx)
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
		priorDelegation.UpdateAmount = priorDelegation.Amount
		priorDelegation.UpdateValidator = req.ValidatorAddress
		priorDelegation.UpdateLocktimeTier = newLocktimeTier

		priorDelegation.State = REDELEGATING
		priorDelegation.LockTime = newLocktime
		priorDelegation.Referrer = req.Referrer
	} else if priorDelegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
		return logDposError(ctx, errors.New("Redelegation amount out of range."), req.String())
	} else {
		// if less than the full amount is being redelegated, create a new
		// delegation for new validator and unbond from former validator
		priorDelegation.State = REDELEGATING
		priorDelegation.UpdateAmount.Value.Sub(&priorDelegation.Amount.Value, &req.Amount.Value)
		priorDelegation.UpdateValidator = priorDelegation.Validator

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
			Referrer:     req.Referrer,
		}
		if err := SetDelegation(ctx, delegation); err != nil {
			return err
		}

		// Emit event for the new delegation
		if err := c.emitDelegatorRedelegatesEvent(ctx, delegation); err != nil {
			return err
		}
	}

	if err := SetDelegation(ctx, priorDelegation); err != nil {
		return err
	}

	return c.emitDelegatorRedelegatesEvent(ctx, priorDelegation)
}

func (c *DPOS) ConsolidateDelegations(ctx contract.Context, req *ConsolidateDelegationsRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 ConsolidateDelegations", "delegator", delegator, "request", req)

	// Unless considation is for the limbo validator, check that the new
	// validator address corresponds to one of the registered candidates
	if loom.UnmarshalAddressPB(req.ValidatorAddress).Compare(LimboValidatorAddress(ctx)) != 0 {
		candidate := GetCandidate(ctx, loom.UnmarshalAddressPB(req.ValidatorAddress))
		// Delegations can only be made to existing candidates
		if candidate == nil {
			return logDposError(ctx, errCandidateNotFound, req.String())
		}
	}

	newDelegation, consolidatedDelegations, unconsolidatedDelegationsCount, err := consolidateDelegations(ctx, req.ValidatorAddress, delegator.MarshalPB())
	if err != nil {
		return err
	}

	return c.emitDelegatorConsolidatesEvent(ctx, newDelegation, consolidatedDelegations, unconsolidatedDelegationsCount)
}

// returns the number of delegations which were not consolidated in the event there is no error
// NOTE: Consolidate delegations is supposed to clear referrer field. If
// a delegator redelegates (to increase locktime reward, for example), this
// redelegation will likely be done via a wallet and thus a wallet can still
// insert its referrer id into the referrer field during redelegation.
func consolidateDelegations(ctx contract.Context, validator, delegator *types.Address) (*Delegation, []*Delegation, int, error) {
	// cycle through all delegations and delete those which are BONDED and
	// unlocked while accumulating their amounts
	delegations, err := returnMatchingDelegations(ctx, validator, delegator)
	if err != nil {
		return nil, nil, -1, err
	}

	unconsolidatedDelegationsCount := 0
	totalDelegationAmount := common.BigZero()
	var consolidatedDelegations []*Delegation
	for _, delegation := range delegations {
		if delegation.LockTime > uint64(ctx.Now().Unix()) || delegation.State != BONDED {
			unconsolidatedDelegationsCount++
			continue
		}

		totalDelegationAmount.Add(totalDelegationAmount, &delegation.Amount.Value)
		consolidatedDelegations = append(consolidatedDelegations, delegation)

		if err = DeleteDelegation(ctx, delegation); err != nil {
			return nil, nil, -1, err
		}
	}

	index, err := GetNextDelegationIndex(ctx, *validator, *delegator)
	if err != nil {
		return nil, nil, -1, err
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
		return nil, nil, -1, err
	}

	return delegation, consolidatedDelegations, unconsolidatedDelegationsCount, nil
}

/// Returns the total amount which will be available to the user's balance
/// if they claim all rewards that are owed to them
func (c *DPOS) CheckRewardsFromAllValidators(ctx contract.StaticContext, req *CheckDelegatorRewardsRequest) (*CheckDelegatorRewardsResponse, error) {
	if req.Delegator == nil {
		return nil, logStaticDposError(ctx, errors.New("CheckRewardsFromAllValidators called with req.Delegator == nil"), req.String())
	}
	delegator := req.Delegator
	validators, err := ValidatorList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	total := big.NewInt(0)
	chainID := ctx.Block().ChainID
	for _, v := range validators {
		valAddress := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(v.PubKey)}
		delegation, err := GetDelegation(ctx, REWARD_DELEGATION_INDEX, *valAddress.MarshalPB(), *delegator)
		if err == contract.ErrNotFound {
			// Skip reward delegations that were not found.
			continue
		} else if err != nil {
			return nil, err
		}

		// Add to the sum
		total.Add(total, delegation.Amount.Value.Int)
	}

	amount := loom.NewBigUInt(total)
	return &CheckDelegatorRewardsResponse{
		Amount: &types.BigUInt{Value: *amount},
	}, nil
}

/// This unbonds the full amount of the rewards delegation from all validators
/// and returns the total amount which will be available to the
func (c *DPOS) ClaimRewardsFromAllValidators(ctx contract.Context, req *ClaimDelegatorRewardsRequest) (*ClaimDelegatorRewardsResponse, error) {
	delegator := ctx.Message().Sender
	validators, err := ValidatorList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	total := big.NewInt(0)
	chainID := ctx.Block().ChainID
	var claimedFromValidators []*types.Address
	var amounts []*types.BigUInt
	for _, v := range validators {
		valAddress := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(v.PubKey)}
		delegation, err := GetDelegation(ctx, REWARD_DELEGATION_INDEX, *valAddress.MarshalPB(), *delegator.MarshalPB())
		if err == contract.ErrNotFound {
			// Skip reward delegations that were not found.
			continue
		} else if err != nil {
			return nil, err
		}

		claimedFromValidators = append(claimedFromValidators, valAddress.MarshalPB())
		amounts = append(amounts, delegation.Amount)

		// Set to UNBONDING and UpdateAmount == Amount, to fully unbond it.
		delegation.State = UNBONDING
		delegation.UpdateAmount = delegation.Amount

		if err := SetDelegation(ctx, delegation); err != nil {
			return nil, err
		}

		err = c.emitDelegatorUnbondsEvent(ctx, delegation)
		if err != nil {
			return nil, err
		}

		// Add to the sum
		total.Add(total, delegation.Amount.Value.Int)
	}

	amount := &types.BigUInt{Value: *loom.NewBigUInt(total)}

	err = c.emitDelegatorClaimsRewardsEvent(ctx, delegator.MarshalPB(), claimedFromValidators, amounts, amount)
	if err != nil {
		return nil, err
	}

	return &ClaimDelegatorRewardsResponse{
		Amount: amount,
	}, nil
}

func (c *DPOS) Unbond(ctx contract.Context, req *UnbondRequest) error {
	delegator := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 Unbond", "delegator", delegator, "request", req)

	if req.ValidatorAddress == nil {
		return logDposError(ctx, errors.New("Unbond called with req.ValidatorAddress == nil"), req.String())
	} else if req.Amount == nil {
		return logDposError(ctx, errors.New("Unbond called with req.Amount == nil"), req.String())
	}

	delegation, err := GetDelegation(ctx, req.Index, *req.ValidatorAddress, *delegator.MarshalPB())
	if err == contract.ErrNotFound {
		return logDposError(ctx, errors.New(fmt.Sprintf("delegation not found: %s %s", req.ValidatorAddress, delegator.MarshalPB())), req.String())
	} else if err != nil {
		return err
	}

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}
	instantUnlock := state.Params.ElectionCycleLength == 0 && delegation.LocktimeTier == TIER_ZERO

	if delegation.Amount.Value.Cmp(&req.Amount.Value) < 0 {
		return logDposError(ctx, errors.New("Unbond amount exceeds delegation amount."), req.String())
	} else if delegation.LockTime > uint64(ctx.Now().Unix()) && !instantUnlock {
		return logDposError(ctx, errors.New("Delegation currently locked."), req.String())
	} else if delegation.State != BONDED {
		return logDposError(ctx, errors.New("Existing delegation not in BONDED state."), req.String())
	} else {
		delegation.State = UNBONDING
		// if req.Amount == 0, the full amount is unbonded
		if common.IsZero(req.Amount.Value) {
			delegation.UpdateAmount = delegation.Amount
		} else {
			delegation.UpdateAmount = req.Amount
		}
		SetDelegation(ctx, delegation)
	}

	return c.emitDelegatorUnbondsEvent(ctx, delegation)
}

func (c *DPOS) CheckDelegation(ctx contract.StaticContext, req *CheckDelegationRequest) (*CheckDelegationResponse, error) {
	ctx.Logger().Debug("DPOSv3 CheckDelegation", "request", req)

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
	ctx.Logger().Debug("DPOSv3 CheckAllDelegations", "request", req)

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
		if loom.UnmarshalAddressPB(d.Delegator).Compare(loom.UnmarshalAddressPB(req.DelegatorAddress)) != 0 {
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
	ctx.Logger().Info("DPOSv3 WhitelistCandidate", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	return c.addCandidateToStatisticList(ctx, req)
}

func (c *DPOS) addCandidateToStatisticList(ctx contract.Context, req *WhitelistCandidateRequest) error {
	_, err := GetStatistic(ctx, loom.UnmarshalAddressPB(req.CandidateAddress))

	tierNumber := req.GetLocktimeTier()
	if tierNumber > 3 {
		return logDposError(ctx, errors.New("Invalid whitelist tier"), req.String())
	}

	if err == contract.ErrNotFound {
		// Creating a ValidatorStatistic entry for candidate with the appropriate
		// lockup period and amount
		SetStatistic(ctx, &ValidatorStatistic{
			Address:         req.CandidateAddress,
			WhitelistAmount: req.Amount,
			LocktimeTier:    tierNumber,
			DelegationTotal: loom.BigZeroPB(),
			SlashPercentage: loom.BigZeroPB(),
		})
	} else if err == nil {
		// ValidatorStatistic must not yet exist for a particular candidate in order
		// to be whitelisted
		return logDposError(ctx, errors.New("Cannot whitelist an already whitelisted candidate."), req.String())
	} else {
		return logDposError(ctx, err, req.String())
	}

	return nil
}

func (c *DPOS) RemoveWhitelistedCandidate(ctx contract.Context, req *RemoveWhitelistedCandidateRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 RemoveWhitelistCandidate", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	statistic, err := GetStatistic(ctx, loom.UnmarshalAddressPB(req.CandidateAddress))
	if err != contract.ErrNotFound && err != nil {
		return err
	}

	if statistic == nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	}
	statistic.UpdateWhitelistAmount = loom.BigZeroPB()
	return SetStatistic(ctx, statistic)
}

func (c *DPOS) ChangeWhitelistInfo(ctx contract.Context, req *ChangeWhitelistInfoRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3", "ChangeWhitelistInfo", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	if req.Amount == nil || !common.IsPositive(req.Amount.Value) {
		return logDposError(ctx, errors.New("Whitelist amount must be a positive number of tokens."), req.String())
	}

	tierNumber := req.GetLocktimeTier()
	if tierNumber > 3 {
		return logDposError(ctx, errors.New("Invalid whitelist tier"), req.String())
	}

	statistic, err := GetStatistic(ctx, loom.UnmarshalAddressPB(req.CandidateAddress))
	if err != nil {
		return logDposError(ctx, errors.New("Candidate is not whitelisted."), req.String())
	}

	statistic.UpdateWhitelistAmount = req.Amount
	statistic.UpdateLocktimeTier = tierNumber

	return SetStatistic(ctx, statistic)
}

func (c *DPOS) RegisterCandidate(ctx contract.Context, req *RegisterCandidateRequest) error {
	candidateAddress := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 RegisterCandidate", "candidate", candidateAddress, "request", req)

	candidates, err := LoadCandidateList(ctx)
	if err != nil {
		return err
	}

	checkAddr := loom.Address{ChainID: ctx.Block().ChainID, Local: loom.LocalAddressFromPublicKey(req.PubKey)}
	if candidateAddress.Compare(checkAddr) != 0 {
		return logDposError(ctx, errors.New("Public key does not match address."), req.String())
	}

	// if candidate record already exists, exit function; candidate record
	// updates are done via the UpdateCandidateRecord function
	cand := candidates.Get(candidateAddress)
	if cand != nil {
		return logDposError(ctx, errCandidateAlreadyRegistered, req.String())
	}

	if err = validateCandidateFee(ctx, req.Fee); err != nil {
		return logDposError(ctx, err, req.String())
	}

	// validate the maximum referral fee candidate is willing to accept
	if err = validateFee(req.MaxReferralPercentage); err != nil {
		return logDposError(ctx, err, req.String())
	}

	// Don't check for an err here because a nil statistic is expected when
	// a candidate registers for the first time
	statistic, _ := GetStatistic(ctx, candidateAddress)

	state, err := LoadState(ctx)
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
			transferFromErr := fmt.Sprintf("Failed coin TransferFrom - registercanidate, %v, %s", candidateAddress.String(), state.Params.RegistrationRequirement.Value.String())
			return logDposError(ctx, err, transferFromErr)
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
		PubKey:                req.PubKey,
		Address:               candidateAddress.MarshalPB(),
		Fee:                   req.Fee,
		NewFee:                req.Fee,
		Name:                  req.Name,
		Description:           req.Description,
		Website:               req.Website,
		State:                 REGISTERED,
		MaxReferralPercentage: req.MaxReferralPercentage,
	}
	candidates.Set(newCandidate)

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateRegistersEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) Unjail(ctx contract.Context, req *UnjailRequest) error {
	if !ctx.FeatureEnabled(features.DPOSVersion3_3, false) {
		return errors.New("DPOS v3.3 is not enabled")
	}

	candidateAddress := ctx.Message().Sender

	// if req.Validator is not nil, make sure that the caller is the oracle
	// only the oracle can unjail other validators, a validator can only unjail itself
	if req.Validator != nil {
		state, err := LoadState(ctx)
		if err != nil {
			return err
		}
		if state.Params.OracleAddress == nil || ctx.Message().Sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
			return errors.New("Only the oracle can unjail other validators")
		}
		candidateAddress = loom.UnmarshalAddressPB(req.Validator)
	}

	statistic, err := GetStatistic(ctx, candidateAddress)
	if err != nil || statistic == nil {
		return errStatisticNotFound
	}

	if !statistic.Jailed {
		return fmt.Errorf("%s is not jailed", candidateAddress.String())
	}

	ctx.Logger().Info("DPOSv3 Unjail", "request", req)
	statistic.Jailed = false
	if err = SetStatistic(ctx, statistic); err != nil {
		return err
	}

	return emitUnjailEvent(ctx, candidateAddress.MarshalPB())
}

func (c *DPOS) ChangeFee(ctx contract.Context, req *ChangeCandidateFeeRequest) error {
	ctx.Logger().Info("DPOSv3 ChangeFee", "request", req)

	candidateAddress := ctx.Message().Sender
	candidates, err := LoadCandidateList(ctx)
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

	if err = validateCandidateFee(ctx, req.Fee); err != nil {
		return logDposError(ctx, err, req.String())
	}

	cand.NewFee = req.Fee
	cand.State = ABOUT_TO_CHANGE_FEE

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return c.emitCandidateFeeChangeEvent(ctx, candidateAddress.MarshalPB(), req.Fee)
}

func (c *DPOS) UpdateCandidateInfo(ctx contract.Context, req *UpdateCandidateInfoRequest) error {
	ctx.Logger().Info("DPOSv3 UpdateCandidateInfo", "request", req)

	candidateAddress := ctx.Message().Sender
	candidates, err := LoadCandidateList(ctx)
	if err != nil {
		return err
	}

	cand := candidates.Get(candidateAddress)
	if cand == nil {
		return errCandidateNotFound
	}

	// validate the maximum referral fee candidate is willing to accept
	if err = validateFee(req.MaxReferralPercentage); err != nil {
		return logDposError(ctx, err, req.String())
	}

	cand.Name = req.Name
	cand.Description = req.Description
	cand.Website = req.Website
	cand.MaxReferralPercentage = req.MaxReferralPercentage

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
	ctx.Logger().Info("DPOSv3 UnregisterCandidate", "candidateAddress", candidateAddress, "request", req)

	candidates, err := LoadCandidateList(ctx)
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
		_, _, lockedDelegations, err := consolidateDelegations(ctx, candidateAddress.MarshalPB(), candidateAddress.MarshalPB())
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

		if err := saveCandidateList(ctx, candidates); err != nil {
			return err
		}

		err = slashValidatorDelegations(ctx, DefaultNoCache, statistic, candidateAddress)
		// NOTE: we ignore the error if DPOSVersion3_4 is not enabled to retain backwards compatibility
		if ctx.FeatureEnabled(features.DPOSVersion3_4, false) {
			if err != nil {
				return err
			}
			if err := SetStatistic(ctx, statistic); err != nil {
				return err
			}
		}
	}

	return c.emitCandidateUnregistersEvent(ctx, candidateAddress.MarshalPB())
}

func (c *DPOS) ListCandidates(ctx contract.StaticContext, req *ListCandidatesRequest) (*ListCandidatesResponse, error) {
	ctx.Logger().Debug("DPOSv3 ListCandidates", "request", req.String())

	candidates, err := LoadCandidateList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	candidateStatistics := make([]*CandidateStatistic, 0)
	for _, candidate := range candidates {
		statistic, err := GetStatistic(ctx, loom.UnmarshalAddressPB(candidate.Address))
		if err != nil && err != contract.ErrNotFound {
			return nil, err
		}

		candidateStatistics = append(candidateStatistics, &CandidateStatistic{
			Candidate: candidate,
			Statistic: statistic,
		})
	}

	return &ListCandidatesResponse{
		Candidates: candidateStatistics,
	}, nil
}

// ***************************
// ELECTIONS & VALIDATORS
// ***************************

// electing and settling rewards settlement
func Elect(ctx contract.Context) error {
	cachedDelegations := &CachedDposStorage{EnableCaching: true}

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// Check if enough time has elapsed to start new validator election
	if state.Params.ElectionCycleLength > (ctx.Now().Unix() - state.LastElectionTime) {
		return nil
	}

	delegationResults, err := rewardAndSlash(ctx, cachedDelegations, state)
	if err != nil {
		return err
	}
	ctx.Logger().Debug("DPOSv3 Elect", "delegationResults", len(delegationResults))

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
					Address:         res.ValidatorAddress.MarshalPB(),
					DelegationTotal: delegationTotal,
					SlashPercentage: loom.BigZeroPB(),
					WhitelistAmount: loom.BigZeroPB(),
				}
			} else {
				statistic.DelegationTotal = delegationTotal

				if statistic.UpdateWhitelistAmount != nil {
					statistic.WhitelistAmount = statistic.UpdateWhitelistAmount
					statistic.LocktimeTier = statistic.UpdateLocktimeTier
					statistic.UpdateWhitelistAmount = nil
				}
			}

			if err = SetStatistic(ctx, statistic); err != nil {
				return err
			}
		}
	}

	// calling `applyPowerCap` ensure that no validator has >28% of the voting
	// power
	if common.IsPositive(*totalValidatorDelegations) {
		state.Validators = applyPowerCap(validators)
		state.LastElectionTime = ctx.Now().Unix()
		state.TotalValidatorDelegations = &types.BigUInt{Value: *totalValidatorDelegations}

		if err = saveState(ctx, state); err != nil {
			return err
		}
	}

	if err = updateCandidateList(ctx); err != nil {
		return err
	}

	ctx.Logger().Debug("DPOSv3 Elect", "Post-Elect State", state)
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

func (c *DPOS) TimeUntilElection(
	ctx contract.StaticContext, req *TimeUntilElectionRequest,
) (*TimeUntilElectionResponse, error) {
	ctx.Logger().Debug("DPOSv3 TimeUntilEleciton", "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return nil, err
	}

	var remainingTime int64
	if state.Params.ElectionCycleLength > 0 {
		remainingTime = state.Params.ElectionCycleLength -
			((ctx.Now().Unix() - state.LastElectionTime) % state.Params.ElectionCycleLength)
	}

	return &TimeUntilElectionResponse{
		TimeUntilElection: remainingTime,
	}, nil
}

func (c *DPOS) ListValidators(ctx contract.StaticContext, req *ListValidatorsRequest) (*ListValidatorsResponse, error) {
	displayStatistics, err := getValidatorStatistics(ctx)
	if err != nil {
		return nil, err
	}
	return &ListValidatorsResponse{
		Statistics: displayStatistics,
	}, nil
}

func ValidatorList(ctx contract.StaticContext) ([]*types.Validator, error) {
	state, err := LoadState(ctx)
	if err != nil {
		return nil, err
	}

	return state.Validators, nil
}

func (c *DPOS) ListDelegations(
	ctx contract.StaticContext, req *ListDelegationsRequest,
) (*ListDelegationsResponse, error) {
	if req.Candidate == nil {
		return nil, logStaticDposError(ctx, errors.New("ListDelegations called with req.Candidate == nil"), req.String())
	}
	return GetCandidateDelegations(ctx, loom.UnmarshalAddressPB(req.Candidate))
}

func (c *DPOS) ListAllDelegations(
	ctx contract.StaticContext, req *ListAllDelegationsRequest,
) (*ListAllDelegationsResponse, error) {
	ctx.Logger().Debug("DPOSv3 ListAllDelegations", "request", req)
	return GetAllDelegations(ctx)
}

// GetCandidateDelegations returns all the current delegations to the given candidate.
func GetCandidateDelegations(ctx contract.StaticContext, candidate loom.Address) (*ListDelegationsResponse, error) {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	total := common.BigZero()
	candidateDelegations := make([]*Delegation, 0)
	for _, d := range delegations {
		if loom.UnmarshalAddressPB(d.Validator).Compare(candidate) != 0 {
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

// GetAllDelegations returns all the current delegations for all candidates.
func GetAllDelegations(ctx contract.StaticContext) (*ListAllDelegationsResponse, error) {
	candidates, err := LoadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*ListDelegationsResponse, 0)
	for _, candidate := range candidates {
		response, err := GetCandidateDelegations(ctx, loom.UnmarshalAddressPB(candidate.Address))
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	return &ListAllDelegationsResponse{
		ListResponses: responses,
	}, nil
}

func (c *DPOS) ListReferrers(ctx contract.StaticContext, req *ListReferrersRequest) (*ListReferrersResponse, error) {
	referrerRange := ctx.Range([]byte(referrerPrefix))
	referrers := make([]*Referrer, 0, len(referrerRange))
	for _, referrer := range referrerRange {
		var addr types.Address
		if err := proto.Unmarshal(referrer.Value, &addr); err != nil {
			return nil, errors.Wrapf(err, "unmarshal referrer %s", string(referrer.Key))
		}
		referrers = append(referrers, &Referrer{
			ReferrerAddress: &addr,
			Name:            string(referrer.Key),
		})
	}
	return &ListReferrersResponse{
		Referrers: referrers,
	}, nil
}

func (c *DPOS) EnableValidatorJailing(ctx contract.Context, req *EnableValidatorJailingRequest) error {
	if !ctx.FeatureEnabled(features.DPOSVersion3_4, false) {
		return errors.New("DPOS v3.4 is not enabled")
	}

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}
	sender := ctx.Message().Sender
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return errOnlyOracle
	}
	if state.Params.JailOfflineValidators == req.JailOfflineValidators {
		return nil
	}

	state.Params.JailOfflineValidators = req.JailOfflineValidators
	return saveState(ctx, state)
}

// ***************************
// REWARDS & SLASHING
// ***************************

func ShiftDowntimeWindow(ctx contract.Context, currentHeight int64, candidates []*Candidate) error {
	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	if state.Params.DowntimePeriod != 0 && (uint64(currentHeight)%state.Params.DowntimePeriod) == 0 {
		ctx.Logger().Info("DPOS ShiftDowntimeWindow", "block", currentHeight)

		downtimeSlashingEnabled := ctx.FeatureEnabled(features.DPOSVersion3_4, false)

		maxDowntimePercentage := defaultMaxDowntimePercentage
		if state.Params.MaxDowntimePercentage != nil {
			maxDowntimePercentage = state.Params.MaxDowntimePercentage.Value
		}

		maximumMissedBlocksBig := CalculateFraction(maxDowntimePercentage, loom.BigUInt{big.NewInt(int64(state.Params.DowntimePeriod))})
		maximumMissedBlocks := maximumMissedBlocksBig.Uint64()

		inactivitySlashPercentage := defaultInactivitySlashPercentage
		if state.Params.CrashSlashingPercentage != nil {
			inactivitySlashPercentage = state.Params.CrashSlashingPercentage.Value
		}

		for _, candidate := range candidates {
			candidateAddress := loom.UnmarshalAddressPB(candidate.Address)
			statistic, err := GetStatistic(ctx, candidateAddress)
			if err != nil {
				if err == contract.ErrNotFound {
					continue
				}
				return err
			}

			if downtimeSlashingEnabled {
				shouldSlash := true
				downtime := getDowntimeRecord(ctx, statistic)
				for i := uint64(0); i < 4; i++ {
					if maximumMissedBlocks >= downtime.Periods[i] {
						shouldSlash = false
						break
					}
				}

				if shouldSlash {
					if err := slash(ctx, statistic, inactivitySlashPercentage); err != nil {
						return err
					}
				}
			}

			statistic.RecentlyMissedBlocks = statistic.RecentlyMissedBlocks << 16
			if err := SetStatistic(ctx, statistic); err != nil {
				return err
			}
		}
	}

	return nil
}

func UpdateDowntimeRecord(ctx contract.Context, downtimePeriod uint64, jailingEnabled bool, validatorAddr loom.Address) error {
	statistic, err := GetStatistic(ctx, validatorAddr)
	if err != nil {
		return logDposError(ctx, err, "UpdateDowntimeRecord attempted to process invalid validator address")
	}

	statistic.RecentlyMissedBlocks = statistic.RecentlyMissedBlocks + 1
	ctx.Logger().Debug(
		"DPOS UpdateDowntimeRecord",
		"validator", statistic.Address,
		"down-blocks", statistic.RecentlyMissedBlocks&0xFFFF,
	)

	// if DPOSv3.3 enabled, jail a valdiator that have been offline for last 4 periods
	jailOfflineValidator := false
	if ctx.FeatureEnabled(features.DPOSVersion3_3, false) {
		jailOfflineValidator = true
	}
	if ctx.FeatureEnabled(features.DPOSVersion3_4, false) {
		jailOfflineValidator = jailingEnabled
	}
	if jailOfflineValidator && !statistic.Jailed {
		downtime := getDowntimeRecord(ctx, statistic)
		if downtime.Periods[0] == downtimePeriod &&
			downtime.Periods[1] == downtimePeriod &&
			downtime.Periods[2] == downtimePeriod &&
			downtime.Periods[3] == downtimePeriod {
			statistic.Jailed = true
			if err := emitJailEvent(ctx, validatorAddr.MarshalPB()); err != nil {
				return err
			}
		}
	}
	return SetStatistic(ctx, statistic)
}

func (c *DPOS) DowntimeRecord(ctx contract.StaticContext, req *DowntimeRecordRequest) (*DowntimeRecordResponse, error) {
	ctx.Logger().Debug("DPOSv3 DowntimeRecord", "request", req)

	downtimeRecords := make([]*DowntimeRecord, 0)
	if req.Validator != nil {
		validator := loom.UnmarshalAddressPB(req.Validator)
		statistic, err := GetStatistic(ctx, validator)
		if err != nil {
			return nil, logStaticDposError(ctx, contract.ErrNotFound, validator.String())
		}
		downtimeRecords = append(downtimeRecords, getDowntimeRecord(ctx, statistic))
	} else {
		validators, err := ValidatorList(ctx)
		if err != nil {
			return nil, logStaticDposError(ctx, err, req.String())
		}

		chainID := ctx.Block().ChainID
		for _, v := range validators {
			validator := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(v.PubKey)}
			statistic, err := GetStatistic(ctx, validator)
			if err != nil {
				return nil, logStaticDposError(ctx, contract.ErrNotFound, validator.String())
			}
			downtimeRecords = append(downtimeRecords, getDowntimeRecord(ctx, statistic))
		}
	}

	state, err := LoadState(ctx)
	if err != nil {
		return nil, err
	}

	return &DowntimeRecordResponse{
		DowntimeRecords: downtimeRecords,
		PeriodLength:    state.Params.DowntimePeriod,
	}, nil
}

func getDowntimeRecord(ctx contract.StaticContext, statistic *ValidatorStatistic) *DowntimeRecord {
	return &DowntimeRecord{
		Validator: statistic.Address,
		Periods: []uint64{
			statistic.RecentlyMissedBlocks & 0xFFFF,
			(statistic.RecentlyMissedBlocks >> 16) & 0xFFFF,
			(statistic.RecentlyMissedBlocks >> 32) & 0xFFFF,
			(statistic.RecentlyMissedBlocks >> 48) & 0xFFFF,
		},
	}
}

func SlashDoubleSign(ctx contract.Context, statistic *ValidatorStatistic) error {
	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	return slash(ctx, statistic, state.Params.ByzantineSlashingPercentage.Value)
}

func slash(ctx contract.Context, statistic *ValidatorStatistic, slashPercentage loom.BigUInt) error {
	updatedAmount := common.BigZero()
	updatedAmount.Add(&statistic.SlashPercentage.Value, &slashPercentage)
	// this check ensures that the slash percentage never exceeds 100%
	if updatedAmount.Cmp(&loom.BigUInt{big.NewInt(hundredPercentInBasisPoints)}) > 0 {
		return nil
	}
	statistic.SlashPercentage = &types.BigUInt{Value: *updatedAmount}

	return emitSlashEvent(ctx, statistic.Address, slashPercentage)
}

// Returns the total amount of tokens which have been distributed to delegators
// and validators as rewards
func (c *DPOS) CheckRewards(ctx contract.StaticContext, req *CheckRewardsRequest) (*CheckRewardsResponse, error) {
	ctx.Logger().Debug("DPOSv3 CheckRewards", "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	return &CheckRewardsResponse{TotalRewardDistribution: state.TotalRewardDistribution}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&DPOS{})

// UTILITIES

func loadCoin(ctx contract.Context) (*ERC20, error) {
	state, err := LoadState(ctx)
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
func rewardAndSlash(ctx contract.Context, cachedDelegations *CachedDposStorage, state *State) ([]*DelegationResult, error) {
	formerValidatorTotals := make(map[string]loom.BigUInt)
	delegatorRewards := make(map[string]*loom.BigUInt)
	distributedRewards := common.BigZero()

	delegations, err := cachedDelegations.loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	for _, validator := range state.Validators {
		candidate := GetCandidateByPubKey(ctx, validator.PubKey)

		if candidate == nil {
			ctx.Logger().Info("Attempted to reward validator no longer on candidates list.", "validator", validator)
			continue
		}

		candidateAddress := loom.UnmarshalAddressPB(candidate.Address)
		validatorKey := candidateAddress.String()
		statistic, _ := GetStatistic(ctx, candidateAddress)

		if statistic == nil {
			delegatorRewards[validatorKey] = common.BigZero()
			formerValidatorTotals[validatorKey] = *common.BigZero()
		} else {
			// If a validator is jailed, don't calculate and distribute rewards
			if ctx.FeatureEnabled(features.DPOSVersion3_3, false) {
				if statistic.Jailed {
					delegatorRewards[validatorKey] = common.BigZero()
					formerValidatorTotals[validatorKey] = *common.BigZero()
					continue
				}
			}
			// If a validator's SlashPercentage is 0, the validator is
			// rewarded for avoiding faults during the last slashing period
			if common.IsZero(statistic.SlashPercentage.Value) {
				distributionTotal := calculateRewards(statistic.DelegationTotal.Value, state.Params, state.TotalValidatorDelegations.Value)

				// The validator share, equal to validator_fee * total_validotor_reward
				// is to be split between the referrers and the validator
				validatorShare := CalculateFraction(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, distributionTotal)

				// delegatorsShare is what fraction of the total rewards will be
				// distributed to delegators
				delegatorsShare := common.BigZero()
				delegatorsShare.Sub(&distributionTotal, &validatorShare)
				delegatorRewards[validatorKey] = delegatorsShare

				// Distribute rewards to referrers
				for _, d := range delegations {
					if loom.UnmarshalAddressPB(d.Validator).Compare(loom.UnmarshalAddressPB(candidate.Address)) == 0 {
						delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
						// if the delegation is not found OR if the delegation
						// has no referrer, we do not need to attempt to
						// distribute the referrer rewards
						if err == contract.ErrNotFound || len(delegation.Referrer) == 0 {
							continue
						} else if err != nil {
							return nil, err
						}

						// if referrer is not found, do not distribute the reward
						referrerAddress := getReferrer(ctx, delegation.Referrer)
						if referrerAddress == nil {
							continue
						}

						// calculate referrerReward
						referrerReward := calculateRewards(delegation.Amount.Value, state.Params, state.TotalValidatorDelegations.Value)
						referrerReward = CalculateFraction(loom.BigUInt{big.NewInt(int64(candidate.Fee))}, referrerReward)
						referrerReward = CalculateFraction(defaultReferrerFee, referrerReward)

						// referrer fees are delegater to limbo validator
						distributedRewards.Add(distributedRewards, &referrerReward)
						cachedDelegations.IncreaseRewardDelegation(ctx, LimboValidatorAddress(ctx).MarshalPB(), referrerAddress, referrerReward)

						// any referrer bonus amount is subtracted from the validatorShare
						validatorShare.Sub(&validatorShare, &referrerReward)
					}
				}

				distributedRewards.Add(distributedRewards, &validatorShare)
				cachedDelegations.IncreaseRewardDelegation(ctx, candidate.Address, candidate.Address, validatorShare)

				// If a validator has some non-zero WhitelistAmount,
				// calculate the validator's reward based on whitelist amount
				if !common.IsZero(statistic.WhitelistAmount.Value) {
					amount := calculateWeightedWhitelistAmount(*statistic)
					whitelistDistribution := calculateShare(amount, statistic.DelegationTotal.Value, *delegatorsShare)
					// increase a delegator's distribution
					distributedRewards.Add(distributedRewards, &whitelistDistribution)
					cachedDelegations.IncreaseRewardDelegation(ctx, candidate.Address, candidate.Address, whitelistDistribution)
				}

				// Keeping track of cumulative distributed rewards by adding
				// every validator's total rewards to
				// `state.TotalRewardDistribution`
				// NOTE: because we round down in every `calculateRewards` call,
				// we expect `state.TotalRewardDistribution` to be a slight
				// overestimate of what was actually distributed. We could be
				// exact with our record keeping by incrementing
				// `state.TotalRewardDistribution` each time
				// `IncreaseRewardDelegation` is called, but because we will not
				// use `state.TotalRewardDistributions` as part of any invariants,
				// we will live with this situation.
				if !ctx.FeatureEnabled(features.DPOSVersion3_1, false) {
					state.TotalRewardDistribution.Value.Add(&state.TotalRewardDistribution.Value, &distributionTotal)
				}
			} else {
				if err := slashValidatorDelegations(ctx, cachedDelegations, statistic, candidateAddress); err != nil {
					return nil, err
				}
				if err := SetStatistic(ctx, statistic); err != nil {
					return nil, err
				}
			}

			formerValidatorTotals[validatorKey] = statistic.DelegationTotal.Value
		}
	}

	newDelegationTotals, err := distributeDelegatorRewards(ctx, cachedDelegations, formerValidatorTotals, delegatorRewards, distributedRewards)
	if err != nil {
		return nil, err
	}

	if ctx.FeatureEnabled(features.DPOSVersion3_1, false) {
		state.TotalRewardDistribution.Value.Add(&state.TotalRewardDistribution.Value, distributedRewards)
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

// returns a Validator's distributionTotal to record the full
// reward amount to be distributed to the validator himself, the delegators and
// the referrers
func calculateRewards(delegationTotal loom.BigUInt, params *Params, totalValidatorDelegations loom.BigUInt) loom.BigUInt {
	cycleSeconds := params.ElectionCycleLength
	reward := CalculateFraction(blockRewardPercentage, delegationTotal)

	// If totalValidator Delegations are high enough to make simple reward
	// calculations result in more rewards given out than the value of `MaxYearlyReward`,
	// scale the rewards appropriately
	yearlyRewardTotal := CalculateFraction(blockRewardPercentage, totalValidatorDelegations)
	if yearlyRewardTotal.Cmp(&params.MaxYearlyReward.Value) > 0 {
		reward.Mul(&reward, &params.MaxYearlyReward.Value)
		reward.Div(&reward, &yearlyRewardTotal)
	}

	// When election cycle = 0, estimate block time at 2 sec
	if cycleSeconds == 0 {
		cycleSeconds = 2
	}
	reward.Mul(&reward, &loom.BigUInt{big.NewInt(cycleSeconds)})
	reward.Div(&reward, &secondsInYear)

	return reward
}

func slashValidatorDelegations(
	ctx contract.Context, cachedDelegations *CachedDposStorage, statistic *ValidatorStatistic,
	validatorAddress loom.Address,
) error {
	if common.IsZero(statistic.SlashPercentage.Value) {
		return nil
	}

	ctx.Logger().Info("DPOSv3 slashValidatorDelegations", "validator", statistic.Address)

	delegations, err := cachedDelegations.loadDelegationList(ctx)
	if err != nil {
		return err
	}

	for _, d := range delegations {
		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return err
		}

		if loom.UnmarshalAddressPB(delegation.Validator).Compare(validatorAddress) == 0 {
			toSlash := CalculateFraction(statistic.SlashPercentage.Value, delegation.Amount.Value)
			updatedAmount := common.BigZero()
			updatedAmount.Sub(&delegation.Amount.Value, &toSlash)
			delegation.Amount = &types.BigUInt{Value: *updatedAmount}
			if err := cachedDelegations.SetDelegation(ctx, delegation); err != nil {
				return err
			}
			if err := emitSlashDelegationEvent(
				ctx, delegation.Delegator, delegation.Validator, delegation.Index, delegation.Amount,
				&types.BigUInt{Value: toSlash}, statistic.SlashPercentage,
			); err != nil {
				return err
			}
		}
	}

	// Slash a whitelisted candidate's whitelist amount. This doesn't affect how
	// much the validator gets back from token timelock, but will decrease the
	// validator's delegation total & thus his ability to earn rewards
	if !common.IsZero(statistic.WhitelistAmount.Value) {
		toSlash := CalculateFraction(statistic.SlashPercentage.Value, statistic.WhitelistAmount.Value)
		beforeSlashedWhitelistAmount := statistic.WhitelistAmount
		updatedAmount := common.BigZero()
		updatedAmount.Sub(&statistic.WhitelistAmount.Value, &toSlash)
		statistic.WhitelistAmount = &types.BigUInt{Value: *updatedAmount}
		if err := emitSlashWhitelistAmountEvent(
			ctx, validatorAddress.MarshalPB(), beforeSlashedWhitelistAmount,
			&types.BigUInt{Value: toSlash}, statistic.SlashPercentage,
		); err != nil {
			return err
		}
	}

	// reset slash total
	statistic.SlashPercentage = loom.BigZeroPB()

	return nil
}

// This function has three goals 1) distribute a validator's rewards to each of
// the delegators, 2) finalize the bonding process for any delegations received
// during the last election period (delegate & unbond calls) and 3) calculate
// the new delegation totals.
func distributeDelegatorRewards(ctx contract.Context, cachedDelegations *CachedDposStorage, formerValidatorTotals map[string]loom.BigUInt, delegatorRewards map[string]*loom.BigUInt, distributedRewards *loom.BigUInt) (map[string]*loom.BigUInt, error) {
	newDelegationTotals := make(map[string]*loom.BigUInt)

	candidates, err := LoadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize delegation totals with whitelist amounts
	for _, candidate := range candidates {
		statistic, _ := GetStatistic(ctx, loom.UnmarshalAddressPB(candidate.Address))

		if statistic != nil && statistic.WhitelistAmount != nil && !common.IsZero(statistic.WhitelistAmount.Value) {
			validatorKey := loom.UnmarshalAddressPB(statistic.Address).String()
			amount := calculateWeightedWhitelistAmount(*statistic)
			newDelegationTotals[validatorKey] = &amount
		}
	}

	delegations, err := cachedDelegations.loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	var currentDelegations = make(DelegationList, len(delegations))
	copy(currentDelegations, delegations)
	for _, d := range currentDelegations {
		delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
		if err == contract.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		validatorKey := loom.UnmarshalAddressPB(delegation.Validator).String()

		// Do not distribute rewards to delegators of the Limbo validator
		// NOTE: because all delegations are sorted in reverse index order, the
		// 0-index delegation (for rewards) is handled last. Therefore, all
		// increases to reward delegations will be reflected in newDelegation
		// totals that are computed at the end of this for loop. (We do this to
		// avoid looping over all delegations twice)
		if loom.UnmarshalAddressPB(delegation.Validator).Compare(LimboValidatorAddress(ctx)) != 0 {
			// allocating validator distributions to delegators
			// based on former validator delegation totals
			delegationTotal := formerValidatorTotals[validatorKey]
			rewardsTotal := delegatorRewards[validatorKey]
			if rewardsTotal != nil {
				weightedDelegation := calculateWeightedDelegationAmount(*delegation)
				delegatorDistribution := calculateShare(weightedDelegation, delegationTotal, *rewardsTotal)
				// increase a delegator's distribution
				distributedRewards.Add(distributedRewards, &delegatorDistribution)
				cachedDelegations.IncreaseRewardDelegation(ctx, delegation.Validator, delegation.Delegator, delegatorDistribution)

				// If the reward delegation is updated by the
				// IncreaseRewardDelegation command, we must be sure to use this
				// updated version in the rest of the loop. No other delegations
				// (non-rewards) have the possibility of being updated outside
				// of this loop.
				if ctx.FeatureEnabled(features.DPOSVersion3_1, false) && d.Index == REWARD_DELEGATION_INDEX {
					delegation, err = GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
					if err == contract.ErrNotFound {
						continue
					} else if err != nil {
						return nil, err
					}
				}
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
				transferFromErr := fmt.Sprintf("Failed coin Transfer - distributeDelegatorRewards, %v, %s", delegation.Delegator.String(), delegation.UpdateAmount.Value.String())
				return nil, logDposError(ctx, err, transferFromErr)
			}
		} else if delegation.State == REDELEGATING {
			if err = cachedDelegations.DeleteDelegation(ctx, delegation); err != nil {
				return nil, err
			}
			delegation.Validator = delegation.UpdateValidator
			delegation.Amount = delegation.UpdateAmount
			delegation.LocktimeTier = delegation.UpdateLocktimeTier

			index, err := GetNextDelegationIndex(ctx, *delegation.Validator, *delegation.Delegator)
			if err != nil {
				return nil, err
			}
			delegation.Index = index

			validatorKey = loom.UnmarshalAddressPB(delegation.Validator).String()
		}

		// Delete any delegation whose full amount has been unbonded. In all
		// other cases, update the delegation state to BONDED and reset its
		// UpdateAmount
		if common.IsZero(delegation.Amount.Value) && delegation.State == UNBONDING {
			if err := cachedDelegations.DeleteDelegation(ctx, delegation); err != nil {
				return nil, err
			}
		} else {
			// After a delegation update, zero out UpdateAmount
			delegation.UpdateAmount = loom.BigZeroPB()
			delegation.State = BONDED

			resetDelegationIfExpired(ctx, delegation)
			if err := cachedDelegations.SetDelegation(ctx, delegation); err != nil {
				return nil, err
			}
		}

		// Calculate delegation totals for all validators except the Limbo
		// validator
		if loom.UnmarshalAddressPB(delegation.Validator).Compare(LimboValidatorAddress(ctx)) != 0 {
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

	ourDelegator := loom.UnmarshalAddressPB(delegator)
	ourValidator := loom.UnmarshalAddressPB(validator)

	var matchingDelegations []*Delegation
	for _, d := range delegations {
		dValidator := loom.UnmarshalAddressPB(d.Validator)
		dDelegator := loom.UnmarshalAddressPB(d.Delegator)
		if dDelegator.Compare(ourDelegator) != 0 || dValidator.Compare(ourValidator) != 0 {
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
	ctx.Logger().Debug("DPOSv3 CheckRewardDelegation", "delegator", delegator, "request", req)

	if req.ValidatorAddress == nil {
		return nil, logStaticDposError(ctx, errors.New("CheckRewardDelegation called with req.ValidatorAddress == nil"), req.String())
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
	ctx.Logger().Debug("DPOSv3 GetState", "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, req.String())
	}

	return &GetStateResponse{State: state}, nil
}

// *************************
// ORACLE METHODS
// *************************

func (c *DPOS) RegisterReferrer(ctx contract.Context, req *RegisterReferrerRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 RegisterReferrer", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	if err := setReferrer(ctx, req.Name, req.Address); err != nil {
		return err
	}
	return c.emitReferrerRegistersEvent(ctx, req.Name, req.Address)
}

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
	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	sender := ctx.Message().Sender
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
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
	ctx.Logger().Info("DPOSv3 SetElectionCycle", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.ElectionCycleLength = req.ElectionCycle

	return saveState(ctx, state)
}

func (c *DPOS) SetDowntimePeriod(ctx contract.Context, req *SetDowntimePeriodRequest) error {
	if !ctx.FeatureEnabled(features.DPOSVersion3_2, false) {
		return errors.New("DPOS v3.2 is not enabled")
	}

	sender := ctx.Message().Sender
	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.DowntimePeriod = req.DowntimePeriod

	return saveState(ctx, state)
}

func (c *DPOS) SetMaxYearlyReward(ctx contract.Context, req *SetMaxYearlyRewardRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetMaxYearlyReward", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.MaxYearlyReward = req.MaxYearlyReward

	return saveState(ctx, state)
}

func (c *DPOS) SetRegistrationRequirement(ctx contract.Context, req *SetRegistrationRequirementRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetRegistrationRequirement", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.RegistrationRequirement = req.RegistrationRequirement

	return saveState(ctx, state)
}

func (c *DPOS) SetValidatorCount(ctx contract.Context, req *SetValidatorCountRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetValidatorCount", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.ValidatorCount = uint64(req.ValidatorCount)

	return saveState(ctx, state)
}

func (c *DPOS) SetOracleAddress(ctx contract.Context, req *SetOracleAddressRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetOracleAddress", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	state.Params.OracleAddress = req.OracleAddress

	return saveState(ctx, state)
}

func (c *DPOS) SetSlashingPercentages(ctx contract.Context, req *SetSlashingPercentagesRequest) error {
	if ctx.FeatureEnabled(features.DPOSVersion3_4, false) {
		if req.CrashSlashingPercentage == nil || req.ByzantineSlashingPercentage == nil {
			return errors.New("slashing percentages must be specified")
		}
	}

	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetSlashingPercentage", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	if req.CrashSlashingPercentage.Value.Cmp(&loom.BigUInt{big.NewInt(hundredPercentInBasisPoints)}) > 0 ||
		req.ByzantineSlashingPercentage.Value.Cmp(&loom.BigUInt{big.NewInt(hundredPercentInBasisPoints)}) > 0 {
		return errors.New("Invalid slashing percentage")
	}

	state.Params.CrashSlashingPercentage = req.CrashSlashingPercentage
	state.Params.ByzantineSlashingPercentage = req.ByzantineSlashingPercentage

	return saveState(ctx, state)
}

func (c *DPOS) SetMaxDowntimePercentage(ctx contract.Context, req *SetMaxDowntimePercentageRequest) error {
	if !ctx.FeatureEnabled(features.DPOSVersion3_4, false) {
		return errors.New("DPOS v3.4 is not enabled")
	}
	if req.MaxDowntimePercentage == nil {
		return logDposError(ctx, errors.New("Must supply value for MaxDowntimePercentage."), req.String())
	}

	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetMaxDowntimePercentage", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	if err := validatePercentage(req.MaxDowntimePercentage.Value); err != nil {
		return logDposError(ctx, err, req.String())
	}

	state.Params.MaxDowntimePercentage = req.MaxDowntimePercentage

	return saveState(ctx, state)
}

func (c *DPOS) SetMinCandidateFee(ctx contract.Context, req *SetMinCandidateFeeRequest) error {
	sender := ctx.Message().Sender
	ctx.Logger().Info("DPOSv3 SetMinCandidateFee", "sender", sender, "request", req)

	state, err := LoadState(ctx)
	if err != nil {
		return err
	}

	//TODO: this will be replaced with voting system next week
	// ensure that function is only executed when called by oracle
	if state.Params.OracleAddress == nil || sender.Compare(loom.UnmarshalAddressPB(state.Params.OracleAddress)) != 0 {
		return logDposError(ctx, errOnlyOracle, req.String())
	}

	if err := validateFee(req.MinCandidateFee); err != nil {
		return logDposError(ctx, err, req.String())
	}

	state.Params.MinCandidateFee = req.MinCandidateFee

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

func emitJailEvent(ctx contract.Context, validator *types.Address) error {
	marshalled, err := proto.Marshal(&DposJailEvent{
		Validator: validator,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, JailEventTopic)
	return nil
}

func emitUnjailEvent(ctx contract.Context, validator *types.Address) error {
	marshalled, err := proto.Marshal(&DposJailEvent{
		Validator: validator,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, UnjailEventTopic)
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

func emitSlashDelegationEvent(
	ctx contract.Context, delegator, validator *types.Address,
	delegationIndex uint64, delegationAmount, slashAmount, slashPercentage *types.BigUInt,
) error {
	marshalled, err := proto.Marshal(&DposSlashDelegationEvent{
		Validator:        validator,
		Delegator:        delegator,
		DelegationAmount: delegationAmount,
		DelegationIndex:  delegationIndex,
		SlashAmount:      slashAmount,
		SlashPercentage:  slashPercentage,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, SlashDelegationEventTopic)
	return nil
}

func emitSlashWhitelistAmountEvent(
	ctx contract.Context, validator *types.Address, whitelistAmount, slashAmount, slashPercentage *types.BigUInt,
) error {
	marshalled, err := proto.Marshal(&DposSlashWhitelistAmountEvent{
		Validator:       validator,
		WhitelistAmount: whitelistAmount,
		SlashAmount:     slashAmount,
		SlashPercentage: slashPercentage,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, SlashWhitelistAmountEventTopic)
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

func (c *DPOS) emitDelegatorDelegatesEvent(ctx contract.Context, delegation *Delegation) error {
	marshalled, err := proto.Marshal(&DposDelegatorDelegatesEvent{
		Delegation: delegation,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorDelegatesEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorRedelegatesEvent(ctx contract.Context, delegation *Delegation) error {
	marshalled, err := proto.Marshal(&DposDelegatorRedelegatesEvent{
		Delegation: delegation,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorRedelegatesEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorConsolidatesEvent(
	ctx contract.Context,
	newDelegation *Delegation,
	consolidatedDelegations []*Delegation,
	unconsolidatedDelegationsCount int) error {
	marshalled, err := proto.Marshal(&DposDelegatorConsolidatesEvent{
		NewDelegation:                  newDelegation,
		ConsolidatedDelegations:        consolidatedDelegations,
		UnconsolidatedDelegationsCount: int64(unconsolidatedDelegationsCount),
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorConsolidatesEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorUnbondsEvent(ctx contract.Context, delegation *Delegation) error {
	marshalled, err := proto.Marshal(&DposDelegatorUnbondsEvent{
		Delegation: delegation,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, DelegatorUnbondsEventTopic)
	return nil
}

func (c *DPOS) emitReferrerRegistersEvent(ctx contract.Context, name string, address *types.Address) error {
	marshalled, err := proto.Marshal(&DposReferrerRegistersEvent{
		Name:    name,
		Address: address,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, ReferrerRegistersEventTopic)
	return nil
}

func (c *DPOS) emitDelegatorClaimsRewardsEvent(ctx contract.Context, delegator *types.Address, validators []*types.Address, amounts []*types.BigUInt, total *types.BigUInt) error {
	marshalled, err := proto.Marshal(&DposDelegatorClaimsRewardsEvent{
		Delegator:           delegator,
		Validators:          validators,
		Amounts:             amounts,
		TotalRewardsClaimed: total,
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

// In case of migration from v2, this should be called in place of Init() and it
// should be the first call made to the dposv3 Contract. The call should be made
// from within the `Dump` function of the dposv2 contract.
func Initialize(ctx contract.Context, initState *InitializationState) error {
	ctx.Logger().Info("DPOSv3 Initialize")

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

func getValidatorStatistics(ctx contract.StaticContext) ([]*ValidatorStatistic, error) {
	validators, err := ValidatorList(ctx)
	if err != nil {
		return nil, logStaticDposError(ctx, err, "")
	}

	chainID := ctx.Block().ChainID

	displayStatistics := make([]*ValidatorStatistic, 0)
	for _, validator := range validators {
		address := loom.Address{ChainID: chainID, Local: loom.LocalAddressFromPublicKey(validator.PubKey)}

		// get validator statistics
		stat, _ := GetStatistic(ctx, address)
		if stat == nil {
			stat = &ValidatorStatistic{
				Address: address.MarshalPB(),
			}
		}
		displayStatistics = append(displayStatistics, stat)
	}

	return displayStatistics, nil
}

// TotalStaked computes the total amount of LOOM staked on-chain, including whitelisted amounts
// locked on Ethereum, but excluding any whitelisted amounts on bootstrap nodes.
func TotalStaked(ctx contract.StaticContext, bootstrapNodes map[string]bool) (*types.BigUInt, error) {
	validatorStats, err := getValidatorStatistics(ctx)
	if err != nil {
		return nil, err
	}
	statistics := map[string]*ValidatorStatistic{}
	for _, statistic := range validatorStats {
		nodeAddr := loom.UnmarshalAddressPB(statistic.Address)
		if _, ok := bootstrapNodes[strings.ToLower(nodeAddr.String())]; !ok {
			statistics[statistic.Address.String()] = statistic
		}
	}

	candidateList := map[string]bool{}
	candidates, err := LoadCandidateList(ctx)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		candidateAddr := loom.UnmarshalAddressPB(candidate.Address)
		if _, ok := bootstrapNodes[strings.ToLower(candidateAddr.String())]; !ok {
			candidateList[candidate.Address.String()] = true
		}
	}

	delegationList, err := loadDelegationList(ctx)
	if err != nil {
		return nil, err
	}

	totalStaked := &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)}
	// Sum all delegations
	for _, d := range delegationList {
		if _, ok := candidateList[d.Validator.String()]; ok {
			delegation, err := GetDelegation(ctx, d.Index, *d.Validator, *d.Delegator)
			if err == contract.ErrNotFound {
				continue
			} else if err != nil {
				return nil, err
			}
			totalStaked.Value.Add(&totalStaked.Value, &delegation.Amount.Value)
		}
	}
	// Sum all whitelist amounts of validators except bootstrap validators
	for _, candidate := range candidates {
		if statistic, ok := statistics[candidate.Address.String()]; ok {
			if statistic.WhitelistAmount != nil {
				totalStaked.Value.Add(&totalStaked.Value, &statistic.WhitelistAmount.Value)
			}
		}
	}
	return totalStaked, nil
}
