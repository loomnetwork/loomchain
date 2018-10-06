// +build evm

package gateway

import (
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	levm "github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/pkg/errors"
)

// Returns all unclaimed tokens for an account
func unclaimedTokensByOwner(ctx contract.StaticContext, ownerAddr loom.Address) ([]*UnclaimedToken, error) {
	result := []*UnclaimedToken{}
	ownerKey := unclaimedTokensRangePrefix(ownerAddr)
	for _, entry := range ctx.Range(ownerKey) {
		var unclaimedToken UnclaimedToken
		if err := proto.Unmarshal(entry.Value, &unclaimedToken); err != nil {
			return nil, errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
		result = append(result, &unclaimedToken)
	}
	return result, nil
}

// Returns all unclaimed tokens for a token contract
func unclaimedTokenDepositorsByContract(ctx contract.StaticContext, tokenAddr loom.Address) ([]loom.Address, error) {
	result := []loom.Address{}
	contractKey := unclaimedTokenDepositorsRangePrefix(tokenAddr)
	for _, entry := range ctx.Range(contractKey) {
		var addr types.Address
		if err := proto.Unmarshal(entry.Value, &addr); err != nil {
			return nil, errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
		result = append(result, loom.UnmarshalAddressPB(&addr))
	}
	return result, nil
}

func genERC721Deposits(tokenAddr, owner loom.Address, blocks []uint64, values [][]int64) []*MainnetEvent {
	if len(values) > 0 && len(values) != len(blocks) {
		panic("insufficent number of values")
	}
	result := []*MainnetEvent{}
	for i, b := range blocks {
		numTokens := 5
		if len(values) > 0 {
			numTokens = len(values[i])
		}
		for j := 0; j < numTokens; j++ {
			tokenID := loom.NewBigUIntFromInt(int64(j + 1))
			if len(values) > 0 {
				tokenID = loom.NewBigUIntFromInt(values[i][j])
			}
			result = append(result, &MainnetEvent{
				EthBlock: b,
				Payload: &MainnetDepositEvent{
					Deposit: &MainnetTokenDeposited{
						TokenKind:     TokenKind_ERC721,
						TokenContract: tokenAddr.MarshalPB(),
						TokenOwner:    owner.MarshalPB(),
						TokenID:       &types.BigUInt{Value: *tokenID},
					},
				},
			})
		}
	}
	return result
}

func genERC20Deposits(tokenAddr, owner loom.Address, blocks []uint64, values []int64) []*MainnetEvent {
	if len(values) != len(blocks) {
		panic("insufficent number of values")
	}
	result := []*MainnetEvent{}
	for i, b := range blocks {
		result = append(result, &MainnetEvent{
			EthBlock: b,
			Payload: &MainnetDepositEvent{
				Deposit: &MainnetTokenDeposited{
					TokenKind:     TokenKind_ERC20,
					TokenContract: tokenAddr.MarshalPB(),
					TokenOwner:    owner.MarshalPB(),
					TokenAmount:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(values[i])},
				},
			},
		})
	}
	return result
}

type erc721xToken struct {
	ID     int64
	Amount int64
}

// Returns a list of ERC721X deposit events, and the total amount per token ID.
func genERC721XDeposits(
	tokenAddr, owner loom.Address, blocks []uint64, tokens [][]*erc721xToken,
) ([]*MainnetEvent, []*erc721xToken) {
	totals := map[int64]int64{}
	result := []*MainnetEvent{}
	for i, b := range blocks {
		for _, token := range tokens[i] {
			result = append(result, &MainnetEvent{
				EthBlock: b,
				Payload: &MainnetDepositEvent{
					Deposit: &MainnetTokenDeposited{
						TokenKind:     TokenKind_ERC721X,
						TokenContract: tokenAddr.MarshalPB(),
						TokenOwner:    owner.MarshalPB(),
						TokenID:       &types.BigUInt{Value: *loom.NewBigUIntFromInt(token.ID)},
						TokenAmount:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(token.Amount)},
					},
				},
			})
			totals[token.ID] = totals[token.ID] + token.Amount
		}
	}

	tokenTotals := []*erc721xToken{}
	for k, v := range totals {
		tokenTotals = append(tokenTotals, &erc721xToken{
			ID:     k,
			Amount: v,
		})
	}
	return result, tokenTotals
}

type testAddressMapperContract struct {
	Contract *address_mapper.AddressMapper
	Address  loom.Address
}

func (am *testAddressMapperContract) AddIdentityMapping(ctx *plugin.FakeContextWithEVM, from, to loom.Address, sig []byte) error {
	return am.Contract.AddIdentityMapping(
		contract.WrapPluginContext(ctx.WithAddress(am.Address)),
		&address_mapper.AddIdentityMappingRequest{
			From:      from.MarshalPB(),
			To:        to.MarshalPB(),
			Signature: sig,
		})
}

func deployAddressMapperContract(ctx *plugin.FakeContextWithEVM) (*testAddressMapperContract, error) {
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

func (gc *testGatewayContract) ContractCtx(ctx *plugin.FakeContextWithEVM) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(gc.Address))
}

func (gc *testGatewayContract) AddContractMapping(ctx *plugin.FakeContextWithEVM, foreignContractAddr, localContractAddr loom.Address) error {
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

func deployGatewayContract(ctx *plugin.FakeContextWithEVM, genesis *InitRequest) (*testGatewayContract, error) {
	gwContract := &Gateway{}
	gwAddr := ctx.CreateContract(contract.MakePluginContract(gwContract))
	gwCtx := contract.WrapPluginContext(ctx.WithAddress(gwAddr))

	err := gwContract.Init(gwCtx, genesis)
	return &testGatewayContract{
		Contract: gwContract,
		Address:  gwAddr,
	}, err
}

type testETHContract struct {
	Contract *ethcoin.ETHCoin
	Address  loom.Address
}

func deployETHContract(ctx *plugin.FakeContextWithEVM) (*testETHContract, error) {
	ethContract := &ethcoin.ETHCoin{}
	contractAddr := ctx.CreateContract(contract.MakePluginContract(ethContract))
	contractCtx := contract.WrapPluginContext(ctx.WithAddress(contractAddr))

	err := ethContract.Init(contractCtx, &ethcoin.InitRequest{})
	return &testETHContract{
		Contract: ethContract,
		Address:  contractAddr,
	}, err
}

func (ec *testETHContract) ContractCtx(ctx *plugin.FakeContextWithEVM) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(ec.Address))
}

func (ec *testETHContract) mintToGateway(ctx *plugin.FakeContextWithEVM, amount *big.Int) error {
	return ec.Contract.MintToGateway(ec.ContractCtx(ctx), &ethcoin.MintToGatewayRequest{
		Amount: &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	})
}

func (ec *testETHContract) approve(ctx *plugin.FakeContextWithEVM, spender loom.Address, amount *big.Int) error {
	return ec.Contract.Approve(ec.ContractCtx(ctx), &ethcoin.ApproveRequest{
		Spender: spender.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	})
}

func (ec *testETHContract) transfer(ctx *plugin.FakeContextWithEVM, to loom.Address, amount *big.Int) error {
	return ec.Contract.Transfer(ec.ContractCtx(ctx), &ethcoin.TransferRequest{
		To:     to.MarshalPB(),
		Amount: &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	})
}

func deployTokenContract(ctx *plugin.FakeContextWithEVM, filename string, gateway, caller loom.Address) (loom.Address, error) {
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

	vm := levm.NewLoomVm(ctx.State, nil, nil, nil)
	_, contractAddr, err = vm.Create(caller, byteCode, loom.NewBigUIntFromInt(0))
	if err != nil {
		return contractAddr, err
	}
	ctx.RegisterContract("", contractAddr, caller)
	return contractAddr, nil
}
