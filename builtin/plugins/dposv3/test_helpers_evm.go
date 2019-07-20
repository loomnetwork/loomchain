// +build evm

package dposv3

import (
	"context"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	cmn "github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	levm "github.com/loomnetwork/loomchain/evm"
	abci "github.com/tendermint/tendermint/abci/types"
)

type testDPOSContractEVM struct {
	Contract *DPOS
	Address  loom.Address
	Ctx      *FakeContextWithEVM
}

// Contract context for tests that need both Go & EVM contracts.
type FakeContextWithEVM1 struct {
	*FakeContextWithEVM
	State                    loomchain.State
	useAccountBalanceManager bool
}

func CreateFakeContextWithEVM1(caller, address loom.Address) *FakeContextWithEVM {
	block := abci.Header{
		ChainID: "default",
		Height:  int64(34),
		Time:    time.Unix(123456789, 0),
	}
	ctx := plugin.CreateFakeContext(caller, address).WithBlock(
		types.BlockHeader{
			ChainID: block.ChainID,
			Height:  block.Height,
			Time:    block.Time.Unix(),
		},
	)
	state := loomchain.NewStoreState(context.Background(), ctx, block, nil, nil)
	return &FakeContextWithEVM{
		FakeContext: ctx,
		State:       state,
	}
}

func (c *FakeContextWithEVM1) WithValidators(validators []*types.Validator) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext:              c.FakeContext.WithValidators(validators),
		State:                    c.State,
		useAccountBalanceManager: c.useAccountBalanceManager,
	}
}

func (c *FakeContextWithEVM1) WithBlock(header loom.BlockHeader) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext:              c.FakeContext.WithBlock(header),
		State:                    c.State,
		useAccountBalanceManager: c.useAccountBalanceManager,
	}
}

func (c *FakeContextWithEVM1) WithSender(caller loom.Address) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext:              c.FakeContext.WithSender(caller),
		State:                    c.State,
		useAccountBalanceManager: c.useAccountBalanceManager,
	}
}

func (c *FakeContextWithEVM1) WithAddress(addr loom.Address) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext:              c.FakeContext.WithAddress(addr),
		State:                    c.State,
		useAccountBalanceManager: c.useAccountBalanceManager,
	}
}

func (c *FakeContextWithEVM1) WithFeature(name string, value bool) *FakeContextWithEVM {
	c.State.SetFeature(name, value)
	return &FakeContextWithEVM{
		FakeContext:              c.FakeContext,
		State:                    c.State,
		useAccountBalanceManager: c.useAccountBalanceManager,
	}
}

func (c *FakeContextWithEVM1) WithAccountBalanceManager(enable bool) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext:              c.FakeContext,
		State:                    c.State,
		useAccountBalanceManager: enable,
	}
}

func (c *FakeContextWithEVM1) AccountBalanceManager(readOnly bool) levm.AccountBalanceManager {
	/*
		ethCoinAddr, err := c.Resolve("ethcoin")
		if err != nil {
			panic(err)
		}
	*/
	return nil
}

func (c *FakeContextWithEVM1) CallEVM(addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
	var createABM levm.AccountBalanceManagerFactoryFunc
	if c.useAccountBalanceManager {
		createABM = c.AccountBalanceManager
	}
	vm := levm.NewLoomVm(c.State, nil, nil, createABM, false)
	return vm.Call(c.ContractAddress(), addr, input, value)
}

func (c *FakeContextWithEVM1) StaticCallEVM(addr loom.Address, input []byte) ([]byte, error) {
	var createABM levm.AccountBalanceManagerFactoryFunc
	if c.useAccountBalanceManager {
		createABM = c.AccountBalanceManager
	}
	vm := levm.NewLoomVm(c.State, nil, nil, createABM, false)
	return vm.StaticCall(c.ContractAddress(), addr, input)
}

func (c *FakeContextWithEVM1) FeatureEnabled(name string, value bool) bool {
	return c.State.FeatureEnabled(name, value)
}

func (c *FakeContextWithEVM1) EnabledFeatures() []string {
	return nil
}

func deployTokenContract(ctx *FakeContextWithEVM, filename string, dpos, caller loom.Address) (loom.Address,
	error) {
	contractAddr := loom.Address{}
	hexByteCode, err := ioutil.ReadFile("contracts/" + filename + ".bin")
	if err != nil {
		return contractAddr, err
	}
	abiBytes, err := ioutil.ReadFile("contracts/" + filename + ".abi")
	if err != nil {
		return contractAddr, err
	}
	contractABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return contractAddr, err
	}
	byteCode := cmn.FromHex(string(hexByteCode))
	// append constructor args to bytecode
	// append constructor args to bytecode
	input, err := contractABI.Pack("")
	if err != nil {
		return contractAddr, err
	}
	byteCode = append(byteCode, input...)

	vm := levm.NewLoomVm(ctx.State, nil, nil, nil, false)
	_, contractAddr, err = vm.Create(caller, byteCode, loom.NewBigUIntFromInt(0))
	if err != nil {
		return contractAddr, err
	}
	ctx.RegisterContract("", contractAddr, caller)
	return contractAddr, nil
}

func deployDPOSContract1(
	ctx *FakeContextWithEVM,
	params *Params,
) (*testDPOSContractEVM, error) {
	dposContract := &DPOS{}
	contractAddr := ctx.CreateContract(contract.MakePluginContract(dposContract))
	dposCtx := ctx.WithAddress(contractAddr)
	contractCtx := contract.WrapPluginContext(dposCtx)

	err := dposContract.Init(contractCtx, &InitRequest{
		Params: params,
		// may also want to set validators
	})

	// Enable the feature flag which enables the reward rounding fix
	ctx.SetFeature(loomchain.DPOSVersion3_1, true)
	ctx.SetFeature(loomchain.DPOSVersion3_6, true)
	return &testDPOSContractEVM{
		Contract: dposContract,
		Address:  contractAddr,
		Ctx:      ctx,
	}, err
}

func (dpos *testDPOSContractEVM) ListAllDelegationsEVM(ctx *FakeContextWithEVM) ([]*ListDelegationsResponse, error) {
	resp, err := dpos.Contract.ListAllDelegations(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ListAllDelegationsRequest{},
	)
	if err != nil {
		return nil, err
	}

	return resp.ListResponses, err
}

func (dpos *testDPOSContractEVM) SetVoucherTokenAddressEVM(ctx *FakeContextWithEVM,
	voucherTokenAddress *loom.Address) error {
	err := dpos.Contract.SetVoucherTokenAddress(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&AddVoucherTokenAddressRequest{VoucherTokenAddress: voucherTokenAddress.MarshalPB()})
	if err != nil {
		return err
	}
	return nil
}

func (dpos *testDPOSContractEVM) ListCandidatesEVM(ctx *FakeContextWithEVM) ([]*CandidateStatistic, error) {
	resp, err := dpos.Contract.ListCandidates(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ListCandidatesRequest{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Candidates, err
}

func (dpos *testDPOSContractEVM) ListValidatorsEVM(ctx *FakeContextWithEVM) ([]*ValidatorStatistic, error) {
	resp, err := dpos.Contract.ListValidators(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ListValidatorsRequest{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Statistics, err
}

func (dpos *testDPOSContractEVM) CheckRewardsEVM(ctx *FakeContextWithEVM) (*common.BigUInt, error) {
	resp, err := dpos.Contract.CheckRewards(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&CheckRewardsRequest{},
	)
	if err != nil {
		return nil, err
	}
	return &resp.TotalRewardDistribution.Value, err
}

func (dpos *testDPOSContractEVM) CheckRewardDelegationEVM(ctx *FakeContextWithEVM, validator *loom.Address) (*Delegation,
	error) {
	resp, err := dpos.Contract.CheckRewardDelegation(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&CheckRewardDelegationRequest{
			ValidatorAddress: validator.MarshalPB(),
		},
	)
	if err != nil {
		return nil, err
	}
	return resp.Delegation, nil
}

func (dpos *testDPOSContractEVM) CheckDelegationEVM(ctx *FakeContextWithEVM, validator *loom.Address,
	delegator *loom.Address) ([]*Delegation, *big.Int, *big.Int, error) {
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
	return resp.Delegations, resp.Amount.Value.Int, resp.WeightedAmount.Value.Int, nil
}

func (dpos *testDPOSContractEVM) DowntimeRecordEVM(ctx *FakeContextWithEVM,
	validator *loom.Address) (*DowntimeRecordResponse, error) {
	var validatorAddr *types.Address
	if validator != nil {
		validatorAddr = validator.MarshalPB()
	}
	resp, err := dpos.Contract.DowntimeRecord(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&DowntimeRecordRequest{
			Validator: validatorAddr,
		},
	)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (dpos *testDPOSContractEVM) CheckAllDelegationsEVM(ctx *FakeContextWithEVM, delegator *loom.Address) ([]*Delegation,
	*big.Int, *big.Int, error) {
	resp, err := dpos.Contract.CheckAllDelegations(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&CheckAllDelegationsRequest{
			DelegatorAddress: delegator.MarshalPB(),
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return resp.Delegations, resp.Amount.Value.Int, resp.WeightedAmount.Value.Int, nil
}

func (dpos *testDPOSContractEVM) RegisterReferrerEVM(ctx *FakeContextWithEVM, referrer loom.Address, name string) error {
	err := dpos.Contract.RegisterReferrer(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&RegisterReferrerRequest{
			Name:    name,
			Address: referrer.MarshalPB(),
		},
	)
	return err
}

func (dpos *testDPOSContractEVM) WhitelistCandidateEVM(ctx *FakeContextWithEVM, candidate loom.Address, amount *big.Int,
	tier LocktimeTier) error {
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

func (dpos *testDPOSContractEVM) ChangeWhitelistInfoEVM(ctx *FakeContextWithEVM, candidate *loom.Address, amount *big.Int,
	tier *LocktimeTier) error {
	req := &ChangeWhitelistInfoRequest{
		CandidateAddress: candidate.MarshalPB(),
		Amount:           &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	}
	if tier != nil {
		req.LocktimeTier = *tier
	}
	err := dpos.Contract.ChangeWhitelistInfo(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		req,
	)
	return err
}

func (dpos *testDPOSContractEVM) ChangeFeeEVM(ctx *FakeContextWithEVM, candidateFee uint64) error {
	err := dpos.Contract.ChangeFee(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ChangeCandidateFeeRequest{
			Fee: candidateFee,
		},
	)
	return err
}

func (dpos *testDPOSContractEVM) RegisterCandidateEVM(
	ctx *FakeContextWithEVM,
	pubKey []byte,
	tier *uint64,
	candidateFee *uint64,
	maxReferralPercentage *uint64,
	candidateName *string,
	candidateDescription *string,
	candidateWebsite *string,
) error {
	req := RegisterCandidateRequest{
		PubKey: pubKey,
	}

	if maxReferralPercentage != nil {
		req.MaxReferralPercentage = *maxReferralPercentage
	}

	if tier != nil {
		req.LocktimeTier = *tier
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

func (dpos *testDPOSContractEVM) UnregisterCandidateEVM(ctx *FakeContextWithEVM) error {
	err := dpos.Contract.UnregisterCandidate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&UnregisterCandidateRequest{},
	)
	return err
}

func (dpos *testDPOSContractEVM) RemoveWhitelistedCandidateEVM(ctx *FakeContextWithEVM, candidate *loom.Address) error {
	err := dpos.Contract.RemoveWhitelistedCandidate(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&RemoveWhitelistedCandidateRequest{CandidateAddress: candidate.MarshalPB()},
	)
	return err
}

func (dpos *testDPOSContractEVM) UnjailEVM(ctx *FakeContextWithEVM, candidate *loom.Address) error {
	var validator *types.Address
	if candidate != nil {
		validator = candidate.MarshalPB()
	}
	err := dpos.Contract.Unjail(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&UnjailRequest{Validator: validator},
	)
	return err
}

func (dpos *testDPOSContractEVM) DelegateEVM(ctx *FakeContextWithEVM, validator *loom.Address, amount *big.Int,
	tier *uint64, referrer *string) error {
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

func (dpos *testDPOSContractEVM) ContractCtxEVM(ctx *FakeContextWithEVM) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(dpos.Address))
}

func (dpos *testDPOSContractEVM) RedelegateEVM(ctx *FakeContextWithEVM, validator *loom.Address,
	newValidator *loom.Address, amount *big.Int, index uint64, tier *uint64, referrer *string) error {
	req := &RedelegateRequest{
		FormerValidatorAddress: validator.MarshalPB(),
		ValidatorAddress:       newValidator.MarshalPB(),
		Index:                  index,
	}

	if amount != nil {
		req.Amount = &types.BigUInt{Value: *loom.NewBigUInt(amount)}
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

func (dpos *testDPOSContractEVM) UnbondEVM(ctx *FakeContextWithEVM, validator *loom.Address, amount *big.Int,
	index uint64) error {
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

func (dpos *testDPOSContractEVM) CheckDelegatorRewardsEVM(ctx *FakeContextWithEVM, delegator *loom.Address) (*big.Int,
	error) {
	claimResponse, err := dpos.Contract.CheckRewardsFromAllValidators(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&CheckDelegatorRewardsRequest{Delegator: delegator.MarshalPB()},
	)
	amt := claimResponse.Amount

	return amt.Value.Int, err
}

func (dpos *testDPOSContractEVM) ClaimDelegatorRewardsEVM(ctx *FakeContextWithEVM) (*big.Int, error) {
	claimResponse, err := dpos.Contract.ClaimRewardsFromAllValidators(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ClaimDelegatorRewardsRequest{},
	)
	amt := claimResponse.Amount

	return amt.Value.Int, err
}

func (dpos *testDPOSContractEVM) ConsolidateDelegationsEVM(ctx *FakeContextWithEVM, validator *loom.Address) error {
	err := dpos.Contract.ConsolidateDelegations(
		contract.WrapPluginContext(ctx.WithAddress(dpos.Address)),
		&ConsolidateDelegationsRequest{
			ValidatorAddress: validator.MarshalPB(),
		},
	)
	return err
}

func (dpos *testDPOSContractEVM) MintVouchersEVM(ctx *FakeContextWithEVM,
	request *MintVoucherRequest) error {
	dposctx := dpos.ContractCtxEVM(ctx.WithAddress(dpos.Address))
	err := dpos.Contract.MintVouchers(dposctx, request)
	return err
}
