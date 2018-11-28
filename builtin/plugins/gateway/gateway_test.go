// +build evm

package gateway

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	lp "github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/plugin"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/stretchr/testify/suite"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	dappAccAddr1 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	ethAccAddr1  = loom.MustParseAddress("eth:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	ethTokenAddr  = loom.MustParseAddress("eth:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	ethTokenAddr2 = loom.MustParseAddress("eth:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	ethTokenAddr3 = loom.MustParseAddress("eth:0x5d1ddf5223a412d24901c32d14ef56cb706c0f64")
)

const (
	coinDecimals = 18
)

type GatewayTestSuite struct {
	suite.Suite
	ethKey    *ecdsa.PrivateKey
	ethKey2   *ecdsa.PrivateKey
	ethAddr   loom.Address
	ethAddr2  loom.Address
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
	ts.ethKey2, err = crypto.GenerateKey()
	require.NoError(err)
	ethLocalAddr, err = loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ts.ethKey2.PublicKey).Hex())
	require.NoError(err)
	ts.ethAddr2 = loom.Address{ChainID: "eth", Local: ethLocalAddr}
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
		lp.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
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
		lp.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
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
	fakeCtx := lp.CreateFakeContext(ts.dAppAddr, loom.RootAddress("chain"))
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
		&ProcessEventBatchRequest{Events: genERC721Deposits(ethTokenAddr, ts.ethAddr, []uint64{5}, nil)},
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
	fakeCtx := lp.CreateFakeContext(ts.dAppAddr, loom.RootAddress("chain"))
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
	fakeCtx := lp.CreateFakeContext(callerAddr, contractAddr)
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
	fakeCtx := plugin.CreateFakeContextWithEVM(ts.dAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/)

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	oracleAddr := ts.dAppAddr
	ownerAddr := ts.dAppAddr2

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	}, false)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr))
	sig, err := address_mapper.SignIdentityMapping(ts.ethAddr, ts.dAppAddr, ts.ethKey)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(fakeCtx, ts.ethAddr, ts.dAppAddr, sig))

	// Batch must have events ordered by block (lowest to highest)
	err = gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx), &ProcessEventBatchRequest{
		Events: genERC721Deposits(ethTokenAddr, ts.ethAddr, []uint64{10, 9}, nil),
	})
	require.Equal(ErrInvalidEventBatch, err, "Should fail because events in batch are out of order")
}

// TODO: Re-enable when ETH transfers are supported
/*
func TestEthDeposit(t *testing.T) {
	callerAddr := addr1
	contractAddr := loom.Address{}
	fakeCtx := lp.CreateFakeContext(callerAddr, contractAddr)
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

func (ts *GatewayTestSuite) TestGatewayERC721Deposit() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContextWithEVM(ts.dAppAddr, loom.RootAddress("chain"))

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ts.dAppAddr2.MarshalPB(),
		Oracles: []*types.Address{ts.dAppAddr.MarshalPB()},
	}, false)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
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
						TokenID:       &types.BigUInt{Value: *loom.NewBigUIntFromInt(123)},
					},
				},
			},
		},
	})
	require.NoError(err)

	erc721 := newERC721StaticContext(gwHelper.ContractCtx(fakeCtx), dappTokenAddr)
	ownerAddr, err := erc721.ownerOf(big.NewInt(123))
	require.NoError(err)
	require.Equal(ts.dAppAddr, ownerAddr)
}

func (ts *GatewayTestSuite) TestWithdrawalRestrictions() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContextWithEVM(ts.dAppAddr, loom.RootAddress("chain"))

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ts.dAppAddr2.MarshalPB(),
		Oracles: []*types.Address{ts.dAppAddr.MarshalPB()},
	}, false)
	require.NoError(err)

	ethHelper, err := deployETHContract(fakeCtx)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr))
	sig, err := address_mapper.SignIdentityMapping(ts.ethAddr, ts.dAppAddr, ts.ethKey)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(fakeCtx, ts.ethAddr, ts.dAppAddr, sig))

	// Mint some tokens/ETH and distribute to users
	token1 := big.NewInt(123)
	token2 := big.NewInt(456)
	token3 := big.NewInt(789)
	ethAmt := big.NewInt(999)
	erc721 := newERC721Context(gwHelper.ContractCtx(fakeCtx), dappTokenAddr)
	require.NoError(erc721.mintToGateway(token1))
	require.NoError(erc721.safeTransferFrom(gwHelper.Address, ts.dAppAddr, token1))
	require.NoError(erc721.mintToGateway(token2))
	require.NoError(erc721.safeTransferFrom(gwHelper.Address, ts.dAppAddr, token2))
	require.NoError(erc721.mintToGateway(token3))
	require.NoError(erc721.safeTransferFrom(gwHelper.Address, ts.dAppAddr2, token3))
	require.NoError(
		ethHelper.mintToGateway(
			fakeCtx.WithSender(gwHelper.Address),
			big.NewInt(0).Mul(ethAmt, big.NewInt(2)),
		),
	)
	require.NoError(ethHelper.transfer(fakeCtx.WithSender(gwHelper.Address), ts.dAppAddr, ethAmt))
	require.NoError(ethHelper.transfer(fakeCtx.WithSender(gwHelper.Address), ts.dAppAddr2, ethAmt))

	// Authorize Gateway to withdraw tokens from users
	erc721 = newERC721Context(
		// Abusing the contract context here, WithAddress() is really meant for contract addresses.
		// Unfortunately WithSender() has no effect when calling the EVM via the fake context
		// because the caller is always set to the contract address stored in the context.
		contract.WrapPluginContext(fakeCtx.WithAddress(ts.dAppAddr)),
		dappTokenAddr,
	)
	require.NoError(erc721.approve(gwHelper.Address, token1))
	require.NoError(erc721.approve(gwHelper.Address, token2))

	erc721 = newERC721Context(
		contract.WrapPluginContext(fakeCtx.WithAddress(ts.dAppAddr2)),
		dappTokenAddr,
	)
	require.NoError(erc721.approve(gwHelper.Address, token3))

	require.NoError(ethHelper.approve(fakeCtx.WithSender(ts.dAppAddr), gwHelper.Address, ethAmt))
	require.NoError(ethHelper.approve(fakeCtx.WithSender(ts.dAppAddr2), gwHelper.Address, ethAmt))

	// Withdraw to an Ethereum account that isn't mapped to a DAppChain account via Address Mapper
	err = gwHelper.Contract.WithdrawToken(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr)),
		&WithdrawTokenRequest{
			TokenContract: ethTokenAddr.MarshalPB(),
			TokenKind:     TokenKind_ERC721,
			TokenID:       &types.BigUInt{Value: *loom.NewBigUInt(token1)},
			Recipient:     ts.ethAddr2.MarshalPB(),
		},
	)
	require.NoError(err)

	// Shouldn't be possible to have more than one pending withdrawal from any one DAppChain account
	err = gwHelper.Contract.WithdrawToken(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr)),
		&WithdrawTokenRequest{
			TokenContract: ethTokenAddr.MarshalPB(),
			TokenKind:     TokenKind_ERC721,
			TokenID:       &types.BigUInt{Value: *loom.NewBigUInt(token2)},
		},
	)
	require.Equal(ErrPendingWithdrawalExists, err)

	// ETH should be treated like any other token, it shouldn't be possible to have more than
	// one pending withdrawal from any one DAppChain account
	err = gwHelper.Contract.WithdrawETH(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr)),
		&WithdrawETHRequest{
			Amount:         &types.BigUInt{Value: *loom.NewBigUInt(ethAmt)},
			MainnetGateway: ethTokenAddr3.MarshalPB(), // doesn't matter for this test
		},
	)
	require.Equal(ErrPendingWithdrawalExists, err)

	// Shouldn't be possible to have more than one pending withdrawal to any one Ethereum account
	err = gwHelper.Contract.WithdrawToken(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr2)),
		&WithdrawTokenRequest{
			TokenContract: ethTokenAddr.MarshalPB(),
			TokenKind:     TokenKind_ERC721,
			TokenID:       &types.BigUInt{Value: *loom.NewBigUInt(token3)},
			Recipient:     ts.ethAddr2.MarshalPB(),
		},
	)
	require.Equal(ErrPendingWithdrawalExists, err)

	// Same restriction should apply to ETH withdrawals
	err = gwHelper.Contract.WithdrawETH(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr2)),
		&WithdrawETHRequest{
			Amount:         &types.BigUInt{Value: *loom.NewBigUInt(ethAmt)},
			MainnetGateway: ethTokenAddr3.MarshalPB(), // doesn't matter for this test
			Recipient:      ts.ethAddr2.MarshalPB(),
		},
	)
	require.Equal(ErrPendingWithdrawalExists, err)

	// Simulate token withdrawal from Ethereum Gateway to clear out the pending withdrawal
	err = gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{
			Events: []*MainnetEvent{
				&MainnetEvent{
					EthBlock: 5,
					Payload: &MainnetWithdrawalEvent{
						Withdrawal: &MainnetTokenWithdrawn{
							TokenOwner:    ts.ethAddr2.MarshalPB(),
							TokenContract: ethTokenAddr.MarshalPB(),
							TokenKind:     TokenKind_ERC721,
							TokenID:       &types.BigUInt{Value: *loom.NewBigUInt(token1)},
						},
					},
				},
			},
		},
	)
	require.NoError(err)

	// Retry the last failed ERC721 withdrawal, should work this time because no pending withdrawal
	// to the Ethereum account should exist...
	err = gwHelper.Contract.WithdrawToken(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr2)),
		&WithdrawTokenRequest{
			TokenContract: ethTokenAddr.MarshalPB(),
			TokenKind:     TokenKind_ERC721,
			TokenID:       &types.BigUInt{Value: *loom.NewBigUInt(token3)},
			Recipient:     ts.ethAddr2.MarshalPB(),
		},
	)
	require.NoError(err)

	// Simulate token withdrawal from Ethereum Gateway to clear out the pending withdrawal
	err = gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{
			Events: []*MainnetEvent{
				&MainnetEvent{
					EthBlock: 10,
					Payload: &MainnetWithdrawalEvent{
						Withdrawal: &MainnetTokenWithdrawn{
							TokenOwner:    ts.ethAddr2.MarshalPB(),
							TokenContract: ethTokenAddr.MarshalPB(),
							TokenKind:     TokenKind_ERC721,
							TokenID:       &types.BigUInt{Value: *loom.NewBigUInt(token3)},
						},
					},
				},
			},
		},
	)
	require.NoError(err)

	// Retry the last failed ETH withdrawal
	err = gwHelper.Contract.WithdrawETH(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr2)),
		&WithdrawETHRequest{
			Amount:         &types.BigUInt{Value: *loom.NewBigUInt(ethAmt)},
			MainnetGateway: ethTokenAddr3.MarshalPB(), // doesn't matter for this test
			Recipient:      ts.ethAddr2.MarshalPB(),
		},
	)
	require.NoError(err)
}

func (ts *GatewayTestSuite) TestReclaimTokensAfterIdentityMapping() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContextWithEVM(ts.dAppAddr, loom.RootAddress("chain"))

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ts.dAppAddr2.MarshalPB(),
		Oracles: []*types.Address{ts.dAppAddr.MarshalPB()},
	}, false)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)
	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr, dappTokenAddr))

	// Don't add the identity mapping between the depositor's Mainnet & DAppChain addresses...

	// Send tokens to Gateway Go contract
	tokensByBlock := [][]int64{
		[]int64{485, 437, 223},
		[]int64{643, 234},
		[]int64{968},
		[]int64{942},
	}
	deposits := genERC721Deposits(
		ethTokenAddr,
		ts.ethAddr,
		[]uint64{5, 9, 11, 13},
		tokensByBlock,
	)

	// None of the tokens will be transferred to their owner because the depositor didn't add an
	// identity mapping
	require.NoError(gwHelper.Contract.ProcessEventBatch(
		gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{Events: deposits}),
	)

	// Since the tokens weren't transferred they shouldn't exist on the DAppChain yet
	erc721 := newERC721StaticContext(gwHelper.ContractCtx(fakeCtx), dappTokenAddr)
	tokenCount := 0
	for _, tokens := range tokensByBlock {
		for _, tokenID := range tokens {
			tokenCount++
			_, err := erc721.ownerOf(big.NewInt(tokenID))
			require.Error(err)
		}
	}
	unclaimedTokens, err := unclaimedTokensByOwner(gwHelper.ContractCtx(fakeCtx), ts.ethAddr)
	require.NoError(err)
	require.Equal(1, len(unclaimedTokens))
	require.Equal(tokenCount, len(unclaimedTokens[0].Amounts))
	depositors, err := unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr)
	require.NoError(err)
	require.Equal(1, len(depositors))

	// The depositor finally add an identity mapping...
	sig, err := address_mapper.SignIdentityMapping(ts.ethAddr, ts.dAppAddr, ts.ethKey)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(fakeCtx, ts.ethAddr, ts.dAppAddr, sig))

	// and attempts to reclaim previously deposited tokens...
	require.NoError(gwHelper.Contract.ReclaimDepositorTokens(
		gwHelper.ContractCtx(fakeCtx),
		&ReclaimDepositorTokensRequest{},
	))

	for _, tokens := range tokensByBlock {
		for _, tokenID := range tokens {
			ownerAddr, err := erc721.ownerOf(big.NewInt(tokenID))
			require.NoError(err)
			require.Equal(ts.dAppAddr, ownerAddr)
		}
	}
	unclaimedTokens, err = unclaimedTokensByOwner(gwHelper.ContractCtx(fakeCtx), ts.ethAddr)
	require.NoError(err)
	require.Equal(0, len(unclaimedTokens))
	depositors, err = unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr)
	require.NoError(err)
	require.Equal(0, len(depositors))
}

func (ts *GatewayTestSuite) TestReclaimTokensAfterContractMapping() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContextWithEVM(ts.dAppAddr, loom.RootAddress("chain"))

	addressMapper, err := deployAddressMapperContract(fakeCtx)
	require.NoError(err)

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ts.dAppAddr2.MarshalPB(),
		Oracles: []*types.Address{ts.dAppAddr.MarshalPB()},
	}, false)
	require.NoError(err)

	// Deploy token contracts to DAppChain EVM
	erc721Addr, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)
	erc20Addr, err := deployTokenContract(fakeCtx, "SampleERC20Token", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)
	erc721xAddr, err := deployTokenContract(fakeCtx, "SampleERC721XToken", gwHelper.Address, ts.dAppAddr)
	require.NoError(err)

	// Don't add the contract mapping between the Mainnet & DAppChain contracts...

	aliceEthAddr := ts.ethAddr
	aliceDAppAddr := ts.dAppAddr
	bobEthAddr := ts.ethAddr2
	bobDAppAddr := ts.dAppAddr2

	sig, err := address_mapper.SignIdentityMapping(aliceEthAddr, aliceDAppAddr, ts.ethKey)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(fakeCtx, aliceEthAddr, aliceDAppAddr, sig))
	sig, err = address_mapper.SignIdentityMapping(bobEthAddr, bobDAppAddr, ts.ethKey2)
	require.NoError(err)
	require.NoError(addressMapper.AddIdentityMapping(
		fakeCtx.WithSender(bobDAppAddr),
		bobEthAddr, bobDAppAddr, sig,
	))

	erc721tokensByBlock := [][]int64{
		[]int64{485, 437, 223},
		[]int64{643, 234},
		[]int64{968},
		[]int64{942},
	}
	erc721deposits := genERC721Deposits(
		ethTokenAddr,
		aliceEthAddr,
		[]uint64{5, 9, 11, 13},
		erc721tokensByBlock,
	)
	erc721tokensByBlock2 := [][]int64{
		[]int64{1485, 1437, 1223},
		[]int64{2643, 2234},
		[]int64{3968},
	}
	erc721deposits2 := genERC721Deposits(
		ethTokenAddr,
		bobEthAddr,
		[]uint64{15, 19, 23},
		erc721tokensByBlock2,
	)
	erc20amountsByBlock := []int64{150, 238, 580}
	erc20deposits := genERC20Deposits(
		ethTokenAddr2,
		aliceEthAddr,
		[]uint64{24, 27, 29},
		erc20amountsByBlock,
	)
	erc20amountsByBlock2 := []int64{389}
	erc20deposits2 := genERC20Deposits(
		ethTokenAddr2,
		bobEthAddr,
		[]uint64{49},
		erc20amountsByBlock2,
	)
	erc721xTokensByBlock := [][]*erc721xToken{
		[]*erc721xToken{
			&erc721xToken{ID: 345, Amount: 20},
			&erc721xToken{ID: 37, Amount: 10},
			&erc721xToken{ID: 40, Amount: 4},
			&erc721xToken{ID: 0, Amount: 15},
		},
		[]*erc721xToken{
			&erc721xToken{ID: 40, Amount: 2},
			&erc721xToken{ID: 345, Amount: 5},
		},
		[]*erc721xToken{
			&erc721xToken{ID: 37, Amount: 3},
			&erc721xToken{ID: 78, Amount: 300},
			&erc721xToken{ID: 0, Amount: 15},
		},
	}
	erc721xDeposits, erc721xTotals := genERC721XDeposits(
		ethTokenAddr3,
		aliceEthAddr,
		[]uint64{54, 58, 61},
		erc721xTokensByBlock,
	)

	// Send tokens to Gateway Go contract...
	// None of the tokens will be transferred to their owners because the contract mapping
	// doesn't exist.
	require.NoError(gwHelper.Contract.ProcessEventBatch(
		gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{Events: erc721deposits}),
	)
	require.NoError(gwHelper.Contract.ProcessEventBatch(
		gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{Events: erc721deposits2}),
	)
	require.NoError(gwHelper.Contract.ProcessEventBatch(
		gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{Events: erc20deposits}),
	)
	require.NoError(gwHelper.Contract.ProcessEventBatch(
		gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{Events: erc20deposits2}),
	)
	require.NoError(gwHelper.Contract.ProcessEventBatch(
		gwHelper.ContractCtx(fakeCtx),
		&ProcessEventBatchRequest{Events: erc721xDeposits}),
	)

	// Since the tokens weren't transferred they shouldn't exist on the DAppChain yet
	erc721 := newERC721StaticContext(gwHelper.ContractCtx(fakeCtx), erc721Addr)
	for _, tokens := range erc721tokensByBlock {
		for _, tokenID := range tokens {
			_, err := erc721.ownerOf(big.NewInt(tokenID))
			require.Error(err)
		}
	}
	for _, tokens := range erc721tokensByBlock2 {
		for _, tokenID := range tokens {
			_, err := erc721.ownerOf(big.NewInt(tokenID))
			require.Error(err)
		}
	}

	erc20 := newERC20StaticContext(gwHelper.ContractCtx(fakeCtx), erc20Addr)
	bal, err := erc20.balanceOf(aliceDAppAddr)
	require.NoError(err)
	require.Equal(int64(0), bal.Int64())
	bal, err = erc20.balanceOf(bobDAppAddr)
	require.NoError(err)
	require.Equal(int64(0), bal.Int64())

	erc721x := newERC721XStaticContext(gwHelper.ContractCtx(fakeCtx), erc721xAddr)
	for _, token := range erc721xTotals {
		bal, err := erc721x.balanceOf(aliceDAppAddr, big.NewInt(token.ID))
		require.NoError(err)
		require.Equal(int64(0), bal.Int64())
	}

	unclaimedTokens, err := unclaimedTokensByOwner(gwHelper.ContractCtx(fakeCtx), aliceEthAddr)
	require.NoError(err)
	require.Equal(3, len(unclaimedTokens))
	unclaimedTokens, err = unclaimedTokensByOwner(gwHelper.ContractCtx(fakeCtx), bobEthAddr)
	require.NoError(err)
	require.Equal(2, len(unclaimedTokens))
	depositors, err := unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr)
	require.NoError(err)
	require.Equal(2, len(depositors))
	depositors, err = unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr2)
	require.NoError(err)
	require.Equal(2, len(depositors))
	depositors, err = unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr3)
	require.NoError(err)
	require.Equal(1, len(depositors))

	// The contract creator finally adds contract mappings...
	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr, erc721Addr))
	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr2, erc20Addr))
	require.NoError(gwHelper.AddContractMapping(fakeCtx, ethTokenAddr3, erc721xAddr))

	// Only the token contract creator should be able to reclaim tokens per contract
	require.Error(gwHelper.Contract.ReclaimContractTokens(
		gwHelper.ContractCtx(fakeCtx.WithSender(ts.dAppAddr3)),
		&ReclaimContractTokensRequest{
			TokenContract: ethTokenAddr.MarshalPB(),
		},
	))
	require.NoError(gwHelper.Contract.ReclaimContractTokens(
		gwHelper.ContractCtx(fakeCtx),
		&ReclaimContractTokensRequest{
			TokenContract: ethTokenAddr.MarshalPB(),
		},
	))
	require.NoError(gwHelper.Contract.ReclaimContractTokens(
		gwHelper.ContractCtx(fakeCtx),
		&ReclaimContractTokensRequest{
			TokenContract: ethTokenAddr2.MarshalPB(),
		},
	))
	require.NoError(gwHelper.Contract.ReclaimContractTokens(
		gwHelper.ContractCtx(fakeCtx),
		&ReclaimContractTokensRequest{
			TokenContract: ethTokenAddr3.MarshalPB(),
		},
	))

	for _, tokens := range erc721tokensByBlock {
		for _, tokenID := range tokens {
			ownerAddr, err := erc721.ownerOf(big.NewInt(tokenID))
			require.NoError(err)
			require.Equal(aliceDAppAddr, ownerAddr)
		}
	}
	for _, tokens := range erc721tokensByBlock2 {
		for _, tokenID := range tokens {
			ownerAddr, err := erc721.ownerOf(big.NewInt(tokenID))
			require.NoError(err)
			require.Equal(bobDAppAddr, ownerAddr)
		}
	}

	expectedBal := int64(0)
	for _, amount := range erc20amountsByBlock {
		expectedBal = expectedBal + amount
	}
	bal, err = erc20.balanceOf(aliceDAppAddr)
	require.NoError(err)
	require.Equal(expectedBal, bal.Int64())

	expectedBal = 0
	for _, amount := range erc20amountsByBlock2 {
		expectedBal = expectedBal + amount
	}
	bal, err = erc20.balanceOf(bobDAppAddr)
	require.NoError(err)
	require.Equal(expectedBal, bal.Int64())

	for _, token := range erc721xTotals {
		bal, err := erc721x.balanceOf(aliceDAppAddr, big.NewInt(token.ID))
		require.NoError(err)
		require.Equal(token.Amount, bal.Int64(), "wrong balance for token %d", token.ID)
	}

	// Check all tokens have been claimed...
	unclaimedTokens, err = unclaimedTokensByOwner(gwHelper.ContractCtx(fakeCtx), aliceEthAddr)
	require.NoError(err)
	require.Equal(0, len(unclaimedTokens))
	unclaimedTokens, err = unclaimedTokensByOwner(gwHelper.ContractCtx(fakeCtx), bobEthAddr)
	require.NoError(err)
	require.Equal(0, len(unclaimedTokens))

	depositors, err = unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr)
	require.NoError(err)
	require.Equal(0, len(depositors))
	depositors, err = unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr2)
	require.NoError(err)
	require.Equal(0, len(depositors))
	depositors, err = unclaimedTokenDepositorsByContract(gwHelper.ContractCtx(fakeCtx), ethTokenAddr3)
	require.NoError(err)
	require.Equal(0, len(depositors))
}

func (ts *GatewayTestSuite) TestGetOracles() {
	require := ts.Require()
	fakeCtx := plugin.CreateFakeContextWithEVM(ts.dAppAddr, loom.RootAddress("chain"))

	ownerAddr := ts.dAppAddr2
	oracleAddr := ts.dAppAddr
	oracle2Addr := ts.dAppAddr3

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	}, false)
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
	fakeCtx := plugin.CreateFakeContextWithEVM(ownerAddr, loom.RootAddress("chain"))

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

	fakeCtx := plugin.CreateFakeContextWithEVM(userAddr, loom.RootAddress("chain"))

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	}, false)
	require.NoError(err)

	// Deploy ERC721 Solidity contract to DAppChain EVM
	dappTokenAddr, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, userAddr)
	require.NoError(err)

	dappTokenAddr2, err := deployTokenContract(fakeCtx, "SampleERC721Token", gwHelper.Address, userAddr)
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

	// Verify pending mappings can't be overwritten
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

// A little sanity check to verify TokenID == 0 doesn't get unmarshalled to TokenID == nil
func (ts *GatewayTestSuite) TestUnclaimedTokenMarshalling() {
	require := ts.Require()

	original := UnclaimedToken{
		TokenKind: TokenKind_ERC721X,
		Amounts: []*TokenAmount{
			&TokenAmount{
				TokenID:     &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
				TokenAmount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
			},
			&TokenAmount{
				TokenID:     &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
				TokenAmount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			},
		},
	}
	bytes, err := proto.Marshal(&original)
	require.NoError(err)

	unmarshalled := &UnclaimedToken{}
	require.NoError(proto.Unmarshal(bytes, unmarshalled))

	require.Equal(original.Amounts[0].TokenID, unmarshalled.Amounts[0].TokenID)
	require.Equal(original.Amounts[1].TokenID, unmarshalled.Amounts[1].TokenID)
	require.Equal(original.Amounts[1].TokenAmount, unmarshalled.Amounts[1].TokenAmount)
}

func (ts *GatewayTestSuite) TestLoomCoinTG() {
	require := ts.Require()

	ownerAddr := ts.dAppAddr
	oracleAddr := ts.dAppAddr2
	userAddr := ts.dAppAddr3
	foreignCreatorAddr := ts.ethAddr

	ethTokenAddr := loom.MustParseAddress("eth:0xb16a379ec18d4093666f8f38b11a3071c920207d")

	fakeCtx := plugin.CreateFakeContextWithEVM(userAddr, loom.RootAddress("chain"))

	loomCoinGwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	}, true)
	require.NoError(err)

	require.EqualError(loomCoinGwHelper.Contract.WithdrawETH(loomCoinGwHelper.ContractCtx(fakeCtx.WithSender(userAddr)), &WithdrawETHRequest{
		Amount:         &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		MainnetGateway: foreignCreatorAddr.MarshalPB(),
	}), ErrInvalidRequest.Error(), "WithdrawEth shouldnt happen in loomcoin TG contract")

	require.EqualError(loomCoinGwHelper.Contract.ProcessEventBatch(loomCoinGwHelper.ContractCtx(fakeCtx.WithSender(oracleAddr)), &ProcessEventBatchRequest{
		Events: genERC721Deposits(ethTokenAddr, ts.ethAddr, []uint64{9, 10}, nil),
	}), ErrInvalidRequest.Error(), "ProcessEventBatch wont entertain events other than loomcoin in loomcoin TG contract")

	require.Nil(loomCoinGwHelper.Contract.ProcessEventBatch(loomCoinGwHelper.ContractCtx(fakeCtx.WithSender(oracleAddr)), &ProcessEventBatchRequest{
		Events: genLoomCoinDeposits(ethTokenAddr, ts.ethAddr, []uint64{9, 10}, []int64{10, 11}),
	}), "ProcessEventBatch should entertain events of loomcoin in loomcoin TG")

	gwHelper, err := deployGatewayContract(fakeCtx, &InitRequest{
		Owner:   ownerAddr.MarshalPB(),
		Oracles: []*types.Address{oracleAddr.MarshalPB()},
	}, false)

	require.Nil(gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx.WithSender(oracleAddr)), &ProcessEventBatchRequest{
		Events: genERC721Deposits(ethTokenAddr, ts.ethAddr, []uint64{9, 10}, nil),
	}), "ProcessEventBatch should entertain events other than loomcoin in TG")

	require.EqualError(gwHelper.Contract.ProcessEventBatch(gwHelper.ContractCtx(fakeCtx.WithSender(oracleAddr)), &ProcessEventBatchRequest{
		Events: genLoomCoinDeposits(ethTokenAddr, ts.ethAddr, []uint64{10, 11}, []int64{10, 11}),
	}), ErrInvalidRequest.Error(), "ProcessEventBatch wont entertain events of loomcoin in TG comtract")

}
