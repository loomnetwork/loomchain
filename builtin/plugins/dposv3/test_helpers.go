package dposv3

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	common "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	// "github.com/loomnetwork/loomchain"
	// "github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

type testDPOSContract struct {
	Contract *DPOS
	Address  loom.Address
}

func deployDPOSContract(
	ctx *plugin.FakeContext,
	validatorCount uint64,
	electionCycleLength *int64,
	coinAddr *loom.Address,
	_maxYearlyReward *loom.BigUInt,
	_registrationRequirement *loom.BigUInt,
	_crashSlashingPercentage *loom.BigUInt,
	_byzantineSlashingPercentage *loom.BigUInt,
	oracleAddr *loom.Address,
) (*testDPOSContract, error) {
	dposContract := &DPOS{}
	contractAddr := ctx.CreateContract(contract.MakePluginContract(dposContract))
	contractCtx := contract.WrapPluginContext(ctx.WithAddress(contractAddr))

	params := &Params{
		ValidatorCount: validatorCount,
	}

	if electionCycleLength != nil {
		params.ElectionCycleLength = *electionCycleLength
	}

	if oracleAddr != nil {
		params.OracleAddress = oracleAddr.MarshalPB()
	}

	if coinAddr != nil {
		params.CoinContractAddress = coinAddr.MarshalPB()
	}

	if _crashSlashingPercentage != nil {
		params.CrashSlashingPercentage = &types.BigUInt{Value: *_crashSlashingPercentage}
	}

	if _byzantineSlashingPercentage != nil {
		params.ByzantineSlashingPercentage = &types.BigUInt{Value: *_byzantineSlashingPercentage}
	}

	if _registrationRequirement != nil {
		params.RegistrationRequirement = &types.BigUInt{Value: *_registrationRequirement}
	}

	if _maxYearlyReward != nil {
		params.MaxYearlyReward = &types.BigUInt{Value: *_maxYearlyReward}
	}

	err := dposContract.Init(contractCtx, &InitRequest{
		Params: params,
		// may also want to set validators
	})

	return &testDPOSContract{
		Contract: dposContract,
		Address:  contractAddr,
	}, err
}

func (dpos *testDPOSContract) ListCandidates(ctx *plugin.FakeContext) ([]*CandidateStatistic, error) {
	resp, err := dpos.Contract.ListCandidates(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ListCandidatesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Candidates, err
}

func (dpos *testDPOSContract) ListValidators(ctx *plugin.FakeContext) ([]*ValidatorStatistic, error) {
	resp, err := dpos.Contract.ListValidators(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ListValidatorsRequest{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Statistics, err
}

func (dpos *testDPOSContract) CheckRewards(ctx *plugin.FakeContext) (*common.BigUInt, error) {
	resp, err := dpos.Contract.CheckRewards(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&CheckRewardsRequest{},
	)
	if err != nil {
		return nil, err
	}
	return &resp.TotalRewardDistribution.Value, err
}

func (dpos *testDPOSContract) CheckDelegation(ctx *plugin.FakeContext, validator *loom.Address, delegator *loom.Address) ([]*Delegation, *common.BigUInt, *common.BigUInt, error) {
	resp, err := dpos.Contract.CheckDelegation(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&CheckDelegationRequest{
			ValidatorAddress: validator.MarshalPB(),
			DelegatorAddress: delegator.MarshalPB(),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return resp.Delegations, &resp.Amount.Value, &resp.WeightedAmount.Value, nil
}

func (dpos *testDPOSContract) WhitelistCandidate(ctx *plugin.FakeContext, candidate loom.Address, amount *big.Int, tier LocktimeTier) error {
	err := dpos.Contract.WhitelistCandidate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&WhitelistCandidateRequest{
			CandidateAddress: candidate.MarshalPB(),
			Amount:           &types.BigUInt{Value: *loom.NewBigUInt(amount)},
			LocktimeTier:     tier,
		},
	)
	return err
}

func (dpos *testDPOSContract) ChangeFee(ctx *plugin.FakeContext, candidateFee uint64) error {
	err := dpos.Contract.ChangeFee(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ChangeCandidateFeeRequest{
			Fee: candidateFee,
		},
	)
	return err
}

func (dpos *testDPOSContract) RegisterCandidate(
	ctx *plugin.FakeContext,
	pubKey []byte,
	candidateFee *uint64,
	candidateName *string,
	candidateDescription *string,
	candidateWebsite *string,
) error {
	req := RegisterCandidateRequest{
		PubKey: pubKey,
	}

	if candidateFee != nil {
		req.Fee = *candidateFee
	}

	if candidateName != nil {
		req.Name = *candidateName
	}

	if candidateDescription != nil {
		req.Description = *candidateDescription
	}

	if candidateWebsite != nil {
		req.Website = *candidateWebsite
	}

	err := dpos.Contract.RegisterCandidate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&req,
	)
	return err
}

func (dpos *testDPOSContract) UnregisterCandidate(ctx *plugin.FakeContext) error {
	err := dpos.Contract.UnregisterCandidate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&UnregisterCandidateRequest{},
	)
	return err
}

func (dpos *testDPOSContract) RemoveWhitelistedCandidate(ctx *plugin.FakeContext, candidate *loom.Address) error {
	err := dpos.Contract.RemoveWhitelistedCandidate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&RemoveWhitelistedCandidateRequest{CandidateAddress: candidate.MarshalPB()},
	)
	return err
}

func (dpos *testDPOSContract) Delegate(ctx *plugin.FakeContext, validator *loom.Address, amount *big.Int, tier *uint64, referrer *string) error {
	req := &DelegateRequest{
		ValidatorAddress: validator.MarshalPB(),
		Amount:           &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	}
	if tier != nil {
		req.LocktimeTier = *tier
	}

	if referrer != nil {
		req.Referrer = *referrer
	}

	err := dpos.Contract.Delegate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		req,
	)
	return err
}

func (dpos *testDPOSContract) Redelegate(ctx *plugin.FakeContext, validator *loom.Address, newValidator *loom.Address, amount *big.Int, index uint64, tier *uint64, referrer *string) error {
	req := &RedelegateRequest{
		FormerValidatorAddress: validator.MarshalPB(),
		ValidatorAddress:       newValidator.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *loom.NewBigUInt(amount)},
		Index:                  index,
	}
	if tier != nil {
		req.NewLocktimeTier = *tier
	}

	if referrer != nil {
		req.Referrer = *referrer
	}

	err := dpos.Contract.Redelegate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		req,
	)
	return err
}

func (dpos *testDPOSContract) Unbond(ctx *plugin.FakeContext, validator *loom.Address, amount *big.Int, index uint64) error {
	err := dpos.Contract.Unbond(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&UnbondRequest{
			ValidatorAddress: validator.MarshalPB(),
			Amount:           &types.BigUInt{Value: *loom.NewBigUInt(amount)},
			Index:            index,
		},
	)
	return err
}
