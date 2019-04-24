package dposv3

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	// common "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	// "github.com/loomnetwork/loomchain"
	// "github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

type testDPOSContract struct {
	Contract *DPOS
	Address  loom.Address
}

func (dpos *testDPOSContract) ContractCtx(ctx *plugin.FakeContext) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(dpos.Address))
}

func deployDPOSContract(
	ctx *plugin.FakeContext,
	validatorCount uint64,
	electionCycleLength int64,
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
		CoinContractAddress: coinAddr.MarshalPB(),
		ValidatorCount:      validatorCount,
		OracleAddress:       oracleAddr.MarshalPB(),
		ElectionCycleLength: electionCycleLength,
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
		dpos.ContractCtx(ctx),
		&ListCandidatesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Candidates, err
}

func (dpos *testDPOSContract) ListValidators(ctx *plugin.FakeContext) ([]*ValidatorStatistic, error) {
	resp, err := dpos.Contract.ListValidators(
		dpos.ContractCtx(ctx),
		&ListValidatorsRequest{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Statistics, err
}

func (dpos *testDPOSContract) WhitelistCandidate(ctx *plugin.FakeContext, candidate loom.Address, amount *big.Int, tier LocktimeTier) error {
	err := dpos.Contract.WhitelistCandidate(
		dpos.ContractCtx(ctx),
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
		dpos.ContractCtx(ctx),
		&ChangeCandidateFeeRequest{
			Fee: candidateFee,
		},
	)
	return err
}

func (dpos *testDPOSContract) RegisterCandidate(
	ctx *plugin.FakeContext,
	pubKey []byte,
	candidateFee uint64,
	candidateName string,
	candidateDescription string,
	candidateWebsite string,
) error {
	err := dpos.Contract.RegisterCandidate(dpos.ContractCtx(ctx),
		&RegisterCandidateRequest{
			PubKey:      pubKey,
			Fee:         candidateFee,
			Name:        candidateName,
			Description: candidateDescription,
			Website:     candidateWebsite,
		},
	)
	return err
}

func (dpos *testDPOSContract) UnregisterCandidate(ctx *plugin.FakeContext) error {
	err := dpos.Contract.UnregisterCandidate(
		dpos.ContractCtx(ctx),
		&UnregisterCandidateRequest{},
	)
	return err
}
