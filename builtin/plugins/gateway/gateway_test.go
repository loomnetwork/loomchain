// +build evm

package gateway

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	levm "github.com/loomnetwork/loomchain/evm"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/abci/types"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")

	dappAccAddr1 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	ethAccAddr1  = loom.MustParseAddress("eth:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	ethTokenAddr = loom.RootAddress("eth:0xb16a379ec18d4093666f8f38b11a3071c920207d")
)

const (
	coinDecimals = 18
)

type GatewayTestSuite struct {
	suite.Suite
	ethKey   *ecdsa.PrivateKey
	ethAddr  loom.Address
	dAppAddr loom.Address
}

func (ts *GatewayTestSuite) SetupTest() {
	require := ts.Require()
	var err error
	ts.ethKey, err = crypto.GenerateKey()
	require.NoError(err)
	ethLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ts.ethKey.PublicKey).Hex())
	require.NoError(err)
	ts.ethAddr = loom.Address{ChainID: "eth", Local: ethLocalAddr}
	ts.dAppAddr = loom.Address{ChainID: "chain", Local: addr1.Local}
}

func TestGatewayTestSuite(t *testing.T) {
	suite.Run(t, new(GatewayTestSuite))
}

func (ts *GatewayTestSuite) TestInit() {
	require := ts.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	gw := &Gateway{}
	require.NoError(gw.Init(ctx, &InitRequest{}))

	resp, err := gw.GetState(ctx, &GatewayStateRequest{})
	require.NoError(err)
	s := resp.State
	ts.Equal(uint64(0), s.LastEthBlock)
}

func (ts *GatewayTestSuite) TestEmptyEventBatchProcessing() {
	require := ts.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	require.NoError(contract.Init(ctx, &InitRequest{
		Oracles: []*types.Address{addr1.MarshalPB()},
	}))

	// Should error out on an empty batch
	require.Error(contract.ProcessEventBatch(ctx, &ProcessEventBatchRequest{}))
}

func (ts *GatewayTestSuite) TestPermissions() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContext(ts.dAppAddr, loom.RootAddress("chain"))

	gwContract := &Gateway{}
	require.NoError(gwContract.Init(contract.WrapPluginContext(fakeCtx), &InitRequest{}))

	err := gwContract.ProcessEventBatch(
		contract.WrapPluginContext(fakeCtx.WithSender(ts.dAppAddr)),
		&ProcessEventBatchRequest{Events: genTokenDeposits(ts.ethAddr, []uint64{5})},
	)
	require.Equal(ErrNotAuthorized, err, "Should fail because caller is not authorized to call ProcessEventBatchRequest")
}

// TODO: Re-enable when ERC20 is supported
/*
func TestOldEventBatchProcessing(t *testing.T) {
	callerAddr := addr1
	contractAddr := loom.Address{}
	fakeCtx := plugin.CreateFakeContext(callerAddr, contractAddr)
	gw := &Gateway{}
	gwAddr := fakeCtx.CreateContract(contract.MakePluginContract(gw))
	gwCtx := contract.WrapPluginContext(fakeCtx.WithAddress(gwAddr))

	coinContract, err := deployCoinContract(fakeCtx, gwAddr, 100000)
	require.Nil(t, err)
	initialGatewayCoinBal := sciNot(100000, coinDecimals)

	err = gw.Init(gwCtx, &GatewayInitRequest{
		Oracles: []*types.Address{addr1.MarshalPB()},
		Tokens: []*GatewayTokenMapping{&GatewayTokenMapping{
			FromToken: ethTokenAddr.MarshalPB(),
			ToToken:   coinContract.Address.MarshalPB(),
		}},
	})
	require.Nil(t, err)

	coinBal, err := coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.Equal(t, initialGatewayCoinBal, coinBal, "gateway account balance should match initial balance")

	err = gw.ProcessEventBatch(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{5}),
	})
	require.Nil(t, err)
	resp, err := gw.GetState(gwCtx, &GatewayStateRequest{})
	require.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)

	coinBal, err = coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.True(t, coinBal.Cmp(initialGatewayCoinBal) < 0, "gateway account balance should have been reduced")

	// Events from each block should only be processed once, even if multiple batches contain the
	// same block.
	err = gw.ProcessEventBatch(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{5}),
	})
	require.NotNil(t, err)
	resp, err = gw.GetState(gwCtx, &GatewayStateRequest{})
	require.Nil(t, err)
	s = resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)

	coinBal2, err := coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.True(t, coinBal.Cmp(coinBal2) == 0, "gateway account balance should not have changed")
}
*/

func (ts *GatewayTestSuite) TestOutOfOrderEventBatchProcessing() {
	require := ts.Require()
	fakeCtx := createFakeContext(ts.dAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/)

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	gwHelper, err := deployGatewayContract(fakeCtx, ts.dAppAddr)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	vm := levm.NewLoomVm(fakeCtx.State, nil)
	dappTokenAddr, err := deployERC721Contract(vm, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	addressMapper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr)
	sig, err := address_mapper.SignIdentityMapping(ts.ethAddr, ts.dAppAddr, ts.ethKey)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(fakeCtx, ts.ethAddr, ts.dAppAddr, sig))

	// Batch must have events ordered by block (lowest to highest)
	err = gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx), &ProcessEventBatchRequest{
		Events: genTokenDeposits(ts.ethAddr, []uint64{10, 9}),
	})
	require.Equal(ErrInvalidEventBatch, err, "Should fail because events in batch are out of order")
}

// TODO: Re-enable when ETH transfers are supported
/*
func TestEthDeposit(t *testing.T) {
	callerAddr := addr1
	contractAddr := loom.Address{}
	fakeCtx := plugin.CreateFakeContext(callerAddr, contractAddr)
	gw := &Gateway{}
	gwAddr := fakeCtx.CreateContract(contract.MakePluginContract(gw))
	gwCtx := contract.WrapPluginContext(fakeCtx.WithAddress(gwAddr))

	coinContract, err := deployCoinContract(fakeCtx, gwAddr, 100000)
	err = gw.Init(gwCtx, &GatewayInitRequest{
		Oracles: []*types.Address{addr1.MarshalPB()},
		Tokens: []*GatewayTokenMapping{&GatewayTokenMapping{
			FromToken: ethTokenAddr.MarshalPB(),
			ToToken:   coinContract.Address.MarshalPB(),
		}},
	})
	require.Nil(t, err)

	bal, err := coinContract.getBalance(fakeCtx, dappAccAddr1)
	require.Nil(t, err)
	assert.Equal(t, uint64(0), bal.Uint64(), "receiver account balance should be zero")

	gwBal, err := coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)

	depositAmount := int64(10)
	err = gw.ProcessEventBatch(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: []*TokenDeposit{
			&TokenDeposit{
				Token:    ethTokenAddr.MarshalPB(),
				From:     ethAccAddr1.MarshalPB(),
				To:       dappAccAddr1.MarshalPB(),
				Amount:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(depositAmount)},
				EthBlock: 5,
			},
		},
	})
	require.Nil(t, err)

	bal2, err := coinContract.getBalance(fakeCtx, dappAccAddr1)
	require.Nil(t, err)
	assert.Equal(t, depositAmount, bal2.Int64(), "receiver account balance should match deposit amount")

	gwBal2, err := coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.Equal(t, depositAmount, gwBal.Sub(gwBal, gwBal2).Int64(), "gateway account balance reduced by deposit amount")
}
*/

func genTokenDeposits(owner loom.Address, blocks []uint64) []*MainnetEvent {
	result := []*MainnetEvent{}
	for _, b := range blocks {
		for i := 0; i < 5; i++ {
			result = append(result, &MainnetEvent{
				EthBlock: b,
				Payload: &MainnetDepositEvent{
					Deposit: &MainnetTokenDeposited{
						TokenKind:     TokenKind_ERC721,
						TokenContract: ethTokenAddr.MarshalPB(),
						TokenOwner:    owner.MarshalPB(),
						Value: &types.BigUInt{
							Value: *loom.NewBigUIntFromInt(int64(i + 1)),
						},
					},
				},
			})
		}
	}
	return result
}

type testCoinContract struct {
	Contract *coin.Coin
	Address  loom.Address
}

/*
func deployCoinContract(ctx *plugin.FakeContext, gwAddr loom.Address, bal uint64) (*testCoinContract, error) {
	// Deploy the coin contract & give the gateway contract a bunch of coins
	coinContract := &coin.Coin{}
	coinAddr := ctx.CreateContract(contract.MakePluginContract(coinContract))
	coinCtx := contract.WrapPluginContext(ctx.WithAddress(coinAddr))

	err := coinContract.Init(coinCtx, &ctypes.InitRequest{
		Accounts: []*coin.InitialAccount{
			&coin.InitialAccount{
				Owner:   gwAddr.MarshalPB(),
				Balance: bal,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &testCoinContract{
		Contract: coinContract,
		Address:  coinAddr,
	}, nil
}

func (c *testCoinContract) getBalance(ctx *plugin.FakeContext, ownerAddr loom.Address) (*loom.BigUInt, error) {
	resp, err := c.Contract.BalanceOf(
		contract.WrapPluginContext(ctx.WithAddress(c.Address)),
		&coin.BalanceOfRequest{Owner: ownerAddr.MarshalPB()},
	)
	if err != nil {
		return loom.NewBigUIntFromInt(0), err
	}
	return &resp.Balance.Value, nil
}

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}
*/

func (ts *GatewayTestSuite) TestGatewayERC721Deposit() {
	require := ts.Require()
	fakeCtx := createFakeContext(ts.dAppAddr, loom.RootAddress("chain"))

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	gwHelper, err := deployGatewayContract(fakeCtx, ts.dAppAddr)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	vm := levm.NewLoomVm(fakeCtx.State, nil)
	dappTokenAddr, err := deployERC721Contract(vm, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	addressMapper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr)
	sig, err := address_mapper.SignIdentityMapping(ts.ethAddr, ts.dAppAddr, ts.ethKey)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(fakeCtx, ts.ethAddr, ts.dAppAddr, sig))

	// Send token to Gateway Go contract
	err = gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx), &ProcessEventBatchRequest{
		Events: []*MainnetEvent{
			&MainnetEvent{
				EthBlock: 5,
				Payload: &MainnetDepositEvent{
					Deposit: &MainnetTokenDeposited{
						TokenKind:     TokenKind_ERC721,
						TokenContract: ethTokenAddr.MarshalPB(),
						TokenOwner:    ts.ethAddr.MarshalPB(),
						Value:         &types.BigUInt{Value: *loom.NewBigUIntFromInt(123)},
					},
				},
			},
		},
	})
	require.NoError(err)

	ownerAddr, err := ownerOfToken(gwHelper.ContractCtx(fakeCtx), dappTokenAddr, big.NewInt(123))
	require.NoError(err)
	require.Equal(ts.dAppAddr, ownerAddr)
}

type testAddressMapperContract struct {
	Contract *address_mapper.AddressMapper
	Address  loom.Address
}

func (am *testAddressMapperContract) AddIdentityMapping(ctx *fakeContext, from, to loom.Address, sig []byte) error {
	return am.Contract.AddIdentityMapping(
		contract.WrapPluginContext(ctx.WithAddress(am.Address)),
		&address_mapper.AddIdentityMappingRequest{
			From:      from.MarshalPB(),
			To:        to.MarshalPB(),
			Signature: sig,
		})
}

func (am *testAddressMapperContract) AddContractMapping(ctx *fakeContext, from, to loom.Address) error {
	return am.Contract.AddContractMapping(
		contract.WrapPluginContext(ctx.WithAddress(am.Address)),
		&address_mapper.AddContractMappingRequest{
			From: from.MarshalPB(),
			To:   to.MarshalPB(),
		})
}

func deployAddressMapperContract(ctx *fakeContext) (*testAddressMapperContract, error) {
	amContract := &address_mapper.AddressMapper{}
	amAddr := ctx.CreateContract(contract.MakePluginContract(amContract))
	amCtx := contract.WrapPluginContext(ctx.WithAddress(amAddr))

	err := amContract.Init(amCtx, &address_mapper.InitRequest{})
	if err != nil {
		return nil, err
	}
	return &testAddressMapperContract{
		Contract: amContract,
		Address:  amAddr,
	}, nil
}

type testGatewayContract struct {
	Contract *Gateway
	Address  loom.Address
}

func (gc *testGatewayContract) ContractCtx(ctx *fakeContext) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(gc.Address))
}

func deployGatewayContract(ctx *fakeContext, oracleAddr loom.Address) (*testGatewayContract, error) {
	gwContract := &Gateway{}
	gwAddr := ctx.CreateContract(contract.MakePluginContract(gwContract))
	gwCtx := contract.WrapPluginContext(ctx.WithAddress(gwAddr))

	err := gwContract.Init(gwCtx, &InitRequest{
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	})
	return &testGatewayContract{
		Contract: gwContract,
		Address:  gwAddr,
	}, err
}

func deployERC721Contract(vm lvm.VM, filename string, gateway, caller loom.Address) (loom.Address, error) {
	contractAddr := loom.Address{}
	hexByteCode, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	if err != nil {
		return contractAddr, err
	}
	abiBytes, err := ioutil.ReadFile("testdata/" + filename + ".abi")
	if err != nil {
		return contractAddr, err
	}
	contractABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return contractAddr, err
	}
	byteCode := common.FromHex(string(hexByteCode))
	// append constructor args to bytecode
	input, err := contractABI.Pack("", common.BytesToAddress(gateway.Local))
	if err != nil {
		return contractAddr, err
	}
	byteCode = append(byteCode, input...)
	_, contractAddr, err = vm.Create(caller, byteCode)
	if err != nil {
		return contractAddr, err
	}
	return contractAddr, nil
}

// Contract context for tests that need both Go & EVM contracts.
type fakeContext struct {
	*plugin.FakeContext
	State loomchain.State
}

func createFakeContext(caller, address loom.Address) *fakeContext {
	block := abci.Header{
		ChainID: "chain",
		Height:  int64(34),
		Time:    int64(123456789),
	}
	ctx := plugin.CreateFakeContext(caller, address).WithBlock(
		types.BlockHeader{
			ChainID: block.ChainID,
			Height:  block.Height,
			Time:    block.Time,
		},
	)
	state := loomchain.NewStoreState(context.Background(), ctx, block)
	return &fakeContext{
		FakeContext: ctx,
		State:       state,
	}
}

func (c *fakeContext) WithBlock(header loom.BlockHeader) *fakeContext {
	return &fakeContext{
		FakeContext: c.FakeContext.WithBlock(header),
		State:       c.State,
	}
}

func (c *fakeContext) WithSender(caller loom.Address) *fakeContext {
	return &fakeContext{
		FakeContext: c.FakeContext.WithSender(caller),
		State:       c.State,
	}
}

func (c *fakeContext) WithAddress(addr loom.Address) *fakeContext {
	return &fakeContext{
		FakeContext: c.FakeContext.WithAddress(addr),
		State:       c.State,
	}
}

func (c *fakeContext) CallEVM(addr loom.Address, input []byte) ([]byte, error) {
	vm := levm.NewLoomVm(c.State, nil)
	return vm.Call(c.ContractAddress(), addr, input)
}

func (c *fakeContext) StaticCallEVM(addr loom.Address, input []byte) ([]byte, error) {
	vm := levm.NewLoomVm(c.State, nil)
	return vm.StaticCall(c.ContractAddress(), addr, input)
}
