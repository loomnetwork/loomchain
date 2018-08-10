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
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	levm "github.com/loomnetwork/loomchain/evm"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	dappAccAddr1 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	ethAccAddr1  = loom.MustParseAddress("eth:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	ethTokenAddr = loom.MustParseAddress("eth:0xb16a379ec18d4093666f8f38b11a3071c920207d")
)

const (
	coinDecimals = 18
)

type GatewayTestSuite struct {
	suite.Suite
	ethKey    *ecdsa.PrivateKey
	ethAddr   loom.Address
	dAppAddr  loom.Address
	dAppAddr2 loom.Address
	dAppAddr3 loom.Address
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
	ts.dAppAddr2 = loom.Address{ChainID: "chain", Local: addr2.Local}
	ts.dAppAddr3 = loom.Address{ChainID: "chain", Local: addr3.Local}
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
	require.NoError(gw.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
	}))

	resp, err := gw.GetState(ctx, &GatewayStateRequest{})
	require.NoError(err)
	s := resp.State
	ts.Equal(uint64(0), s.LastMainnetBlockNum)
}

func (ts *GatewayTestSuite) TestEmptyEventBatchProcessing() {
	require := ts.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	require.NoError(contract.Init(ctx, &InitRequest{
		Owner:   addr1.MarshalPB(),
		Oracles: []*types.Address{addr1.MarshalPB()},
	}))

	// Should error out on an empty batch
	require.Error(contract.ProcessEventBatch(ctx, &ProcessEventBatchRequest{}))
}

func (ts *GatewayTestSuite) TestOwnerPermissions() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContext(ts.dAppAddr, loom.RootAddress("chain"))
	ownerAddr := ts.dAppAddr
	oracleAddr := ts.dAppAddr2

	gwContract := &Gateway{}
	require.NoError(gwContract.Init(contract.WrapPluginContext(fakeCtx), &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	}))

	err := gwContract.AddOracle(
		contract.WrapPluginContext(fakeCtx.WithSender(oracleAddr)),
		&AddOracleRequest{Oracle: oracleAddr.MarshalPB()},
	)
	require.Equal(ErrNotAuthorized, err, "Only owner should be allowed to add oracles")

	err = gwContract.RemoveOracle(
		contract.WrapPluginContext(fakeCtx.WithSender(oracleAddr)),
		&RemoveOracleRequest{Oracle: oracleAddr.MarshalPB()},
	)
	require.Equal(ErrNotAuthorized, err, "Only owner should be allowed to remove oracles")

	require.NoError(gwContract.RemoveOracle(
		contract.WrapPluginContext(fakeCtx.WithSender(ownerAddr)),
		&RemoveOracleRequest{Oracle: oracleAddr.MarshalPB()},
	), "Owner should be allowed to remove oracles")

	require.NoError(gwContract.AddOracle(
		contract.WrapPluginContext(fakeCtx.WithSender(ownerAddr)),
		&AddOracleRequest{Oracle: oracleAddr.MarshalPB()},
	), "Owner should be allowed to add oracles")

	err = gwContract.ProcessEventBatch(
		contract.WrapPluginContext(fakeCtx.WithSender(ownerAddr)),
		&ProcessEventBatchRequest{Events: genTokenDeposits(ts.ethAddr, []uint64{5})},
	)
	require.Equal(ErrNotAuthorized, err, "Only an oracle should be allowed to submit Mainnet events")

	err = gwContract.ConfirmWithdrawalReceipt(
		contract.WrapPluginContext(fakeCtx.WithSender(ownerAddr)),
		&ConfirmWithdrawalReceiptRequest{},
	)
	require.Equal(ErrNotAuthorized, err, "Only an oracle should be allowed to confirm withdrawals")
}

func (ts *GatewayTestSuite) TestOraclePermissions() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContext(ts.dAppAddr, loom.RootAddress("chain"))
	ownerAddr := ts.dAppAddr
	oracleAddr := ts.dAppAddr2
	oracle2Addr := ts.dAppAddr3

	gwContract := &Gateway{}
	require.NoError(gwContract.Init(
		contract.WrapPluginContext(fakeCtx),
		&InitRequest{
			Owner:   ownerAddr.MarshalPB(),
			Oracles: []*types.Address{oracleAddr.MarshalPB()},
		},
	))

	// Check that an oracle added via genesis has all the expected permission
	err := gwContract.ProcessEventBatch(
		contract.WrapPluginContext(fakeCtx.WithSender(oracleAddr)),
		&ProcessEventBatchRequest{},
	)
	require.NotEqual(ErrNotAuthorized, err, "Genesis Oracle should be allowed to submit Mainnet events")

	err = gwContract.ConfirmWithdrawalReceipt(
		contract.WrapPluginContext(fakeCtx.WithSender(oracleAddr)),
		&ConfirmWithdrawalReceiptRequest{},
	)
	require.NotEqual(ErrNotAuthorized, err, "Genesis Oracle should be allowed to confirm withdrawals")

	// Check that a newly added oracle has all the expected permissions
	require.NoError(gwContract.AddOracle(
		contract.WrapPluginContext(fakeCtx.WithSender(ownerAddr)),
		&AddOracleRequest{Oracle: oracle2Addr.MarshalPB()},
	))

	err = gwContract.ProcessEventBatch(
		contract.WrapPluginContext(fakeCtx.WithSender(oracle2Addr)),
		&ProcessEventBatchRequest{},
	)
	require.NotEqual(ErrNotAuthorized, err, "New Oracle should be allowed to submit Mainnet events")

	err = gwContract.ConfirmWithdrawalReceipt(
		contract.WrapPluginContext(fakeCtx.WithSender(oracle2Addr)),
		&ConfirmWithdrawalReceiptRequest{},
	)
	require.NotEqual(ErrNotAuthorized, err, "New Oracle should be allowed to confirm withdrawals")

	// Check that an oracle that has been removed had all its permissions revoked
	require.NoError(gwContract.RemoveOracle(
		contract.WrapPluginContext(fakeCtx.WithSender(ownerAddr)),
		&RemoveOracleRequest{Oracle: oracleAddr.MarshalPB()},
	))

	err = gwContract.ProcessEventBatch(
		contract.WrapPluginContext(fakeCtx.WithSender(oracleAddr)),
		&ProcessEventBatchRequest{},
	)
	require.Equal(ErrNotAuthorized, err, "Removed Oracle shouldn't be allowed to submit Mainnet events")

	err = gwContract.ConfirmWithdrawalReceipt(
		contract.WrapPluginContext(fakeCtx.WithSender(oracleAddr)),
		&ConfirmWithdrawalReceiptRequest{},
	)
	require.Equal(ErrNotAuthorized, err, "Removed Oracle shouldn't be allowed to confirm withdrawals")
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

	oracleAddr := ts.dAppAddr
	ownerAddr := ts.dAppAddr2

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	})
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployERC721Contract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr))
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

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ts.dAppAddr2.MarshalPB(),
		Oracles: []*types.Address{ts.dAppAddr.MarshalPB()},
	})
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployERC721Contract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr))
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

func (ts *GatewayTestSuite) TestGetOracles() {
	require := ts.Require()
	fakeCtx := createFakeContext(ts.dAppAddr, loom.RootAddress("chain"))

	ownerAddr := ts.dAppAddr2
	oracleAddr := ts.dAppAddr
	oracle2Addr := ts.dAppAddr3

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	})
	require.NoError(err)

	resp, err := gwHelper.Contract.GetOracles(gwHelper.ContractCtx(fakeCtx), &GetOraclesRequest{})
	require.NoError(err)
	require.Len(resp.Oracles, 1)
	require.Equal(oracleAddr, loom.UnmarshalAddressPB(resp.Oracles[0].Address))

	require.NoError(gwHelper.Contract.AddOracle(
		gwHelper.ContractCtx(fakeCtx.WithSender(ownerAddr)),
		&AddOracleRequest{
			Oracle: oracle2Addr.MarshalPB(),
		},
	))

	resp, err = gwHelper.Contract.GetOracles(gwHelper.ContractCtx(fakeCtx), &GetOraclesRequest{})
	require.NoError(err)
	require.Len(resp.Oracles, 2)
	addr1 := loom.UnmarshalAddressPB(resp.Oracles[0].Address)
	addr2 := loom.UnmarshalAddressPB(resp.Oracles[1].Address)
	if addr1.Compare(oracleAddr) == 0 {
		require.Equal(oracle2Addr, addr2)
	} else if addr2.Compare(oracleAddr) == 0 {
		require.Equal(oracle2Addr, addr1)
	} else {
		require.Fail("unexpected set of oracles")
	}

	require.NoError(gwHelper.Contract.RemoveOracle(
		gwHelper.ContractCtx(fakeCtx.WithSender(ownerAddr)),
		&RemoveOracleRequest{
			Oracle: oracleAddr.MarshalPB(),
		},
	))

	resp, err = gwHelper.Contract.GetOracles(gwHelper.ContractCtx(fakeCtx), &GetOraclesRequest{})
	require.NoError(err)
	require.Len(resp.Oracles, 1)
	require.Equal(oracle2Addr, loom.UnmarshalAddressPB(resp.Oracles[0].Address))
}

func (ts *GatewayTestSuite) TestAddRemoveTokenWithdrawer() {
	require := ts.Require()
	ownerAddr := ts.dAppAddr
	oracleAddr := ts.dAppAddr2
	withdrawerAddr := ts.dAppAddr3
	fakeCtx := createFakeContext(ownerAddr, loom.RootAddress("chain"))

	gwContract := &Gateway{}
	require.NoError(gwContract.Init(
		contract.WrapPluginContext(fakeCtx),
		&InitRequest{
			Owner:   ownerAddr.MarshalPB(),
			Oracles: []*types.Address{oracleAddr.MarshalPB()},
		},
	))
	ctx := contract.WrapPluginContext(fakeCtx)

	s, err := loadState(ctx)
	require.NoError(err)

	require.NoError(addTokenWithdrawer(ctx, s, withdrawerAddr))
	require.Len(s.TokenWithdrawers, 1)
	require.Equal(loom.UnmarshalAddressPB(s.TokenWithdrawers[0]), withdrawerAddr)

	require.NoError(removeTokenWithdrawer(ctx, s, withdrawerAddr))
	require.NoError(err)
	require.Len(s.TokenWithdrawers, 0)
}

func (ts *GatewayTestSuite) TestAddNewContractMapping() {
	require := ts.Require()

	ownerAddr := ts.dAppAddr
	oracleAddr := ts.dAppAddr2
	userAddr := ts.dAppAddr3
	foreignCreatorAddr := ts.ethAddr
	ethTokenAddr := loom.MustParseAddress("eth:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	ethTokenAddr2 := loom.MustParseAddress("eth:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")

	fakeCtx := createFakeContext(userAddr, loom.RootAddress("chain"))

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	})
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployERC721Contract(fakeCtx, "SampleERC721Token", gwHelper.Address, userAddr)
	require.NoError(err)

	dappTokenAddr2, err := deployERC721Contract(fakeCtx, "SampleERC721Token", gwHelper.Address, userAddr)
	require.NoError(err)
	require.NotEqual(dappTokenAddr, dappTokenAddr2)

	hash := ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(ethTokenAddr.Local)),
		ssha.Address(common.BytesToAddress(dappTokenAddr.Local)),
	)

	sig, err := evmcompat.GenerateTypedSig(hash, ts.ethKey, evmcompat.SignatureType_EIP712)
	require.NoError(err)

	// When a user adds a contract mapping a pending contract mapping should be created
	require.NoError(gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr.MarshalPB(),
			LocalContract:             dappTokenAddr.MarshalPB(),
			ForeignContractCreatorSig: sig,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	))

	// Verify pending mappings can't overwritten
	err = gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr.MarshalPB(),
			LocalContract:             dappTokenAddr.MarshalPB(),
			ForeignContractCreatorSig: sig,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	)
	require.Equal(ErrContractMappingExists, err, "AddContractMapping should not allow duplicate mapping")

	hash = ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(ethTokenAddr.Local)),
		ssha.Address(common.BytesToAddress(dappTokenAddr2.Local)),
	)

	sig2, err := evmcompat.GenerateTypedSig(hash, ts.ethKey, evmcompat.SignatureType_EIP712)
	require.NoError(err)

	err = gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr.MarshalPB(),
			LocalContract:             dappTokenAddr2.MarshalPB(),
			ForeignContractCreatorSig: sig2,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	)
	require.Equal(ErrContractMappingExists, err, "AddContractMapping should not allow re-mapping")

	hash = ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(ethTokenAddr2.Local)),
		ssha.Address(common.BytesToAddress(dappTokenAddr.Local)),
	)

	sig3, err := evmcompat.GenerateTypedSig(hash, ts.ethKey, evmcompat.SignatureType_EIP712)
	require.NoError(err)

	err = gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr2.MarshalPB(),
			LocalContract:             dappTokenAddr.MarshalPB(),
			ForeignContractCreatorSig: sig3,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	)
	require.Equal(ErrContractMappingExists, err, "AddContractMapping should not allow re-mapping")

	// Oracle retrieves the tx hash from the pending contract mapping
	unverifiedCreatorsResp, err := gwHelper.Contract.UnverifiedContractCreators(
		gwHelper.ContractCtx(fakeCtx.WithSender(oracleAddr)),
		&UnverifiedContractCreatorsRequest{})
	require.NoError(err)
	require.Len(unverifiedCreatorsResp.Creators, 1)

	// Oracle extracts the contract and creator address from the tx matching the hash, and sends
	// them back to the contract
	require.NoError(gwHelper.Contract.VerifyContractCreators(
		gwHelper.ContractCtx(fakeCtx.WithSender(oracleAddr)),
		&VerifyContractCreatorsRequest{
			Creators: []*VerifiedContractCreator{
				&VerifiedContractCreator{
					ContractMappingID: unverifiedCreatorsResp.Creators[0].ContractMappingID,
					Creator:           foreignCreatorAddr.MarshalPB(),
					Contract:          ethTokenAddr.MarshalPB(),
				},
			},
		}))

	// The contract and creator address provided by the Oracle should match the pending contract
	// mapping so the Gateway contract should've finalized the bi-directionl contract mapping...
	resolvedAddr, err := resolveToLocalContractAddr(
		gwHelper.ContractCtx(fakeCtx.WithSender(gwHelper.Address)),
		ethTokenAddr)
	require.NoError(err)
	require.True(resolvedAddr.Compare(dappTokenAddr) == 0)

	resolvedAddr, err = resolveToForeignContractAddr(
		gwHelper.ContractCtx(fakeCtx.WithSender(gwHelper.Address)),
		dappTokenAddr)
	require.NoError(err)
	require.True(resolvedAddr.Compare(ethTokenAddr) == 0)

	// Verify confirmed mappings can't be overwritten
	err = gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr.MarshalPB(),
			LocalContract:             dappTokenAddr.MarshalPB(),
			ForeignContractCreatorSig: sig,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	)
	require.Equal(ErrContractMappingExists, err, "AddContractMapping should not allow duplicate mapping")

	err = gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr.MarshalPB(),
			LocalContract:             dappTokenAddr2.MarshalPB(),
			ForeignContractCreatorSig: sig2,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	)
	require.Equal(ErrContractMappingExists, err, "AddContractMapping should not allow re-mapping")

	err = gwHelper.Contract.AddContractMapping(
		gwHelper.ContractCtx(fakeCtx.WithSender(userAddr)),
		&AddContractMappingRequest{
			ForeignContract:           ethTokenAddr2.MarshalPB(),
			LocalContract:             dappTokenAddr.MarshalPB(),
			ForeignContractCreatorSig: sig3,
			ForeignContractTxHash:     []byte("0xdeadbeef"),
		},
	)
	require.Equal(ErrContractMappingExists, err, "AddContractMapping should not allow re-mapping")
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

func (gc *testGatewayContract) AddContractMapping(ctx *fakeContext, foreignContractAddr, localContractAddr loom.Address) error {
	contractCtx := gc.ContractCtx(ctx)
	err := contractCtx.Set(contractAddrMappingKey(foreignContractAddr), &ContractAddressMapping{
		From: foreignContractAddr.MarshalPB(),
		To:   localContractAddr.MarshalPB(),
	})
	if err != nil {
		return err
	}
	err = contractCtx.Set(contractAddrMappingKey(localContractAddr), &ContractAddressMapping{
		From: localContractAddr.MarshalPB(),
		To:   foreignContractAddr.MarshalPB(),
	})
	if err != nil {
		return err
	}
	return nil
}

func deployGatewayContract(ctx *fakeContext, genesis *InitRequest) (*testGatewayContract, error) {
	gwContract := &Gateway{}
	gwAddr := ctx.CreateContract(contract.MakePluginContract(gwContract))
	gwCtx := contract.WrapPluginContext(ctx.WithAddress(gwAddr))

	err := gwContract.Init(gwCtx, genesis)
	return &testGatewayContract{
		Contract: gwContract,
		Address:  gwAddr,
	}, err
}

func deployERC721Contract(ctx *fakeContext, filename string, gateway, caller loom.Address) (loom.Address, error) {
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

	vm := levm.NewLoomVm(ctx.State, nil)
	_, contractAddr, err = vm.Create(caller, byteCode)
	if err != nil {
		return contractAddr, err
	}
	ctx.RegisterContract("", contractAddr, caller)
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
