// +build evm

package gateway

import (
	"context"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	levm "github.com/loomnetwork/loomchain/evm"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
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

func TestInit(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	gw := &Gateway{}
	err := gw.Init(ctx, &InitRequest{})
	require.Nil(t, err)

	resp, err := gw.GetState(ctx, &GatewayStateRequest{})
	require.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(0), s.LastEthBlock)
}

func TestEmptyEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &InitRequest{
		Oracles: []*types.Address{addr1.MarshalPB()},
	})
	require.Nil(t, err)

	// Should error out on an empty batch
	err = contract.ProcessEventBatch(ctx, &ProcessEventBatchRequest{})
	require.NotNil(t, err)
}

func TestPermissions(t *testing.T) {
	callerAddr := addr1
	contractAddr := addr1
	fakeCtx := plugin.CreateFakeContext(callerAddr, contractAddr)

	gwContract := &Gateway{}
	err := gwContract.Init(contract.WrapPluginContext(fakeCtx), &InitRequest{})
	require.Nil(t, err)

	err = gwContract.ProcessEventBatch(
		contract.WrapPluginContext(fakeCtx.WithSender(addr2)),
		&ProcessEventBatchRequest{FtDeposits: genTokenDeposits([]uint64{5})},
	)
	require.Equal(t, ErrNotAuthorized, err, "Should fail because caller is not authorized to call ProcessEventBatchRequest")
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

func TestOutOfOrderEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &InitRequest{
		Oracles: []*types.Address{addr1.MarshalPB()},
	})
	require.Nil(t, err)

	// Batch must have events ordered by block (lowest to highest)
	err = contract.ProcessEventBatch(ctx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{10, 9}),
	})
	require.NotNil(t, err)
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

func genTokenDeposits(blocks []uint64) []*TokenDeposit {
	result := []*TokenDeposit{}
	for _, b := range blocks {
		for i := 0; i < 5; i++ {
			result = append(result, &TokenDeposit{
				Token: ethTokenAddr.MarshalPB(),
				From:  ethAccAddr1.MarshalPB(),
				To:    addr2.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *loom.NewBigUIntFromInt(int64(i + 1)),
				},
				EthBlock: b,
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

func TestGatewayERC721Deposit(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	caller := loom.Address{
		ChainID: "default",
		Local:   loom.LocalAddressFromPublicKey(pub[:]),
	}

	fakeCtx := createFakeContext(caller, loom.Address{})
	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(t, err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	vm := levm.NewLoomVm(fakeCtx.State, nil)
	dappTokenAddr, err := deploySolContract(caller, "DAppChainCards", vm)
	require.NoError(t, err)

	addressMapper.AddMapping(fakeCtx, ethTokenAddr, dappTokenAddr)
	addressMapper.AddMapping(fakeCtx, ethAccAddr1, dappAccAddr1)

	// Deploy Gateway Go contract
	gwContract := &Gateway{}
	gwAddr := fakeCtx.CreateContract(contract.MakePluginContract(gwContract))
	gwCtx := contract.WrapPluginContext(fakeCtx.WithAddress(gwAddr))

	err = gwContract.Init(gwCtx, &InitRequest{
		Oracles: []*types.Address{caller.MarshalPB()},
	})
	require.NoError(t, err)

	// Send token to Gateway Go contract
	err = gwContract.ProcessEventBatch(gwCtx, &ProcessEventBatchRequest{
		NftDeposits: []*NFTDeposit{
			&NFTDeposit{
				Token:    ethTokenAddr.MarshalPB(),
				From:     ethAccAddr1.MarshalPB(),
				Uid:      &types.BigUInt{Value: *loom.NewBigUIntFromInt(123)},
				EthBlock: 5,
			},
		},
	})
	require.NoError(t, err)

	ownerAddr, err := ownerOfToken(gwCtx, dappTokenAddr, big.NewInt(123))
	require.NoError(t, err)
	require.Equal(t, ownerAddr, ethAccAddr1)
}

type testAddressMapperContract struct {
	Contract *address_mapper.AddressMapper
	Address  loom.Address
}

func (am *testAddressMapperContract) AddMapping(ctx *fakeContext, from, to loom.Address) error {
	return am.Contract.AddMapping(
		contract.WrapPluginContext(ctx.WithAddress(am.Address)),
		&address_mapper.AddMappingRequest{
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

func deploySolContract(caller loom.Address, filename string, vm lvm.VM) (loom.Address, error) {
	contractAddr := loom.Address{}
	hexByteCode, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	if err != nil {
		return contractAddr, err
	}
	byteCode := common.FromHex(string(hexByteCode))
	if err != nil {
		return contractAddr, err
	}
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
	ctx := plugin.CreateFakeContext(caller, address)
	state := loomchain.NewStoreState(context.Background(), ctx, abci.Header{
		Height: int64(34),
		Time:   int64(123456789),
	})
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
	return vm.Call(c.Message().Sender, addr, input)
}

func (c *fakeContext) StaticCallEVM(addr loom.Address, input []byte) ([]byte, error) {
	vm := levm.NewLoomVm(c.State, nil)
	return vm.StaticCall(c.Message().Sender, addr, input)
}
