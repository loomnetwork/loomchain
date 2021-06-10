// +build evm

package integration_tests

import (
	"io/ioutil"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEthCoinEvmIntegration(t *testing.T) {
	caller := loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	contractAddr := loom.RootAddress("chain")
	fakeCtx := plugin.CreateFakeContextWithEVM(caller, contractAddr)

	ethCoin, err := deployEthCoinContract(fakeCtx)
	require.NoError(t, err)

	testContract, err := deployEthCoinIntegrationTestContract(fakeCtx, caller)
	require.NoError(t, err)

	amount := sciNot(123, 18)
	// give ETH to test contract account via ethcoin
	ethCoin.mint(fakeCtx, testContract.Address, amount)

	// EVM should have updated the ETH balance of the test contract
	balance, err := testContract.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	require.Equal(t, amount.String(), balance.String())

	// EVM should transfer some ETH to the caller address from the test contract address
	require.NoError(t, testContract.withdraw(fakeCtx, amount))
	balance, err = testContract.balance(fakeCtx, caller)
	require.NoError(t, err)
	require.Equal(t, amount.String(), balance.String())

	// transfer ETH back to the contract
	// TODO: do this via testContract.deposit() instead of ethcoin directly
	require.NoError(t, ethCoin.transfer(fakeCtx, caller, testContract.Address, amount))
	balance, err = testContract.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	require.Equal(t, amount.String(), balance.String())

	// If the EVM reverts after ETH is transferred from the test contract to the caller, the
	// transfer itself should not be reverted. This may seem counterintuitive, but assuming the EVM
	// error is propagated outwards properly the node won't persist any changes that occurred in the
	// ethcoin contract anyway.
	require.Error(t, testContract.withdrawThenFail(fakeCtx, amount))
	balance, err = testContract.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	assert.Equal(t, "0", balance.String())
	balance, err = testContract.balance(fakeCtx, caller)
	require.NoError(t, err)
	assert.Equal(t, amount.String(), balance.String())

	// transfer ETH back to the contract
	// TODO: do this via testContract.deposit() instead of ethcoin directly
	require.NoError(t, ethCoin.transfer(fakeCtx, caller, testContract.Address, amount))
	balance, err = testContract.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	require.Equal(t, amount.String(), balance.String())

	// If the EVM reverts before ETH is transferred then there should no change in any balances.
	require.Error(t, testContract.failThenWithdraw(fakeCtx, amount))
	balance, err = testContract.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	assert.Equal(t, amount.String(), balance.String())
	balance, err = testContract.balance(fakeCtx, caller)
	require.NoError(t, err)
	assert.Equal(t, "0", balance.String())

	// TODO: test deposit & transfer in integration test contract

	// Test contract self-destruction
	balanceCallerBefore, err := testContract.balance(fakeCtx, caller)
	require.NoError(t, err)
	balanceContractBefore, err := testContract.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	require.NoError(t, testContract.destroyContract(fakeCtx, caller))

	// Need to create new contract as we just destroyed the old one
	testContract2, err := deployEthCoinIntegrationTestContract(fakeCtx, caller)
	require.NoError(t, err)

	// The contracts balance should now be added to the caller's balance
	balancedCallerAfter, err := testContract2.balance(fakeCtx, caller)
	require.NoError(t, err)
	var expectedBalance big.Int
	expectedBalance.Add(balanceCallerBefore, balanceContractBefore)
	assert.Equal(t, 0, expectedBalance.Cmp(balancedCallerAfter))
	balanceContractAfter, err := testContract2.balance(fakeCtx, testContract.Address)
	require.NoError(t, err)
	assert.Equal(t, "0", balanceContractAfter.String())

}

// Wraps the ethcoin Go contract that holds all the ETH
type ethCoinTestHelper struct {
	Contract *ethcoin.ETHCoin
	Address  loom.Address
}

func (c *ethCoinTestHelper) contractCtx(ctx *plugin.FakeContextWithEVM) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(c.Address))
}

func deployEthCoinContract(ctx *plugin.FakeContextWithEVM) (*ethCoinTestHelper, error) {
	ethCoin := &ethcoin.ETHCoin{}
	addr := ctx.CreateContract(contract.MakePluginContract(ethCoin))
	ethCoinCtx := contract.WrapPluginContext(ctx.WithAddress(addr))

	err := ethCoin.Init(ethCoinCtx, &ethcoin.InitRequest{})
	return &ethCoinTestHelper{
		Contract: ethCoin,
		Address:  addr,
	}, err
}

func (c *ethCoinTestHelper) mint(
	ctx *plugin.FakeContextWithEVM, to loom.Address, amount *big.Int,
) error {
	return ethcoin.Mint(c.contractCtx(ctx), to, loom.NewBigUInt(amount))
}

func (c *ethCoinTestHelper) transfer(
	ctx *plugin.FakeContextWithEVM, from, to loom.Address, amount *big.Int,
) error {
	return ethcoin.Transfer(c.contractCtx(ctx), from, to, loom.NewBigUInt(amount))
}

// Wraps the EthCoinIntegrationTest EVM contract
type ethCoinIntegrationTestHelper struct {
	contractABI *abi.ABI

	Address loom.Address
}

//nolint:unused
func (c *ethCoinIntegrationTestHelper) contractCtx(ctx *plugin.FakeContextWithEVM) contract.Context {
	return contract.WrapPluginContext(ctx.WithAddress(c.Address))
}

func deployEthCoinIntegrationTestContract(
	ctx *plugin.FakeContextWithEVM, caller loom.Address,
) (*ethCoinIntegrationTestHelper, error) {
	contractName := "EthCoinIntegrationTest"
	abiBytes, err := ioutil.ReadFile("testdata/" + contractName + ".abi")
	if err != nil {
		return nil, err
	}
	contractABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, err
	}

	addr, err := deployContractToEVM(ctx, contractName, caller)
	if err != nil {
		return nil, err
	}

	return &ethCoinIntegrationTestHelper{
		contractABI: &contractABI,
		Address:     addr,
	}, nil
}

func (c *ethCoinIntegrationTestHelper) withdraw(ctx *plugin.FakeContextWithEVM, amount *big.Int) error {
	return c.callEVM(ctx, "withdraw", amount)
}

func (c *ethCoinIntegrationTestHelper) withdrawThenFail(ctx *plugin.FakeContextWithEVM, amount *big.Int) error {
	return c.callEVM(ctx, "withdrawThenFail", amount)
}

func (c *ethCoinIntegrationTestHelper) failThenWithdraw(ctx *plugin.FakeContextWithEVM, amount *big.Int) error {
	return c.callEVM(ctx, "failThenWithdraw", amount)
}

func (c *ethCoinIntegrationTestHelper) balance(ctx *plugin.FakeContextWithEVM, owner loom.Address) (*big.Int, error) {
	ownerAddr := common.BytesToAddress(owner.Local)
	var result *big.Int
	if err := c.staticCallEVM(ctx, "getBalance", &result, ownerAddr); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ethCoinIntegrationTestHelper) destroyContract(ctx *plugin.FakeContextWithEVM, receiver loom.Address) error {
	receiverAddr := common.BytesToAddress(receiver.Local)
	return c.callEVM(ctx, "destroyContract", receiverAddr)
}

func (c *ethCoinIntegrationTestHelper) callEVM(
	ctx *plugin.FakeContextWithEVM, method string, params ...interface{},
) error {
	input, err := c.contractABI.Pack(method, params...)
	if err != nil {
		return err
	}
	vm := evm.NewLoomVm(ctx.State, nil, nil, ctx.AccountBalanceManager, nil, false)
	_, err = vm.Call(ctx.Message().Sender, c.Address, input, loom.NewBigUIntFromInt(0))
	if err != nil {
		return err
	}
	return nil
}

func (c *ethCoinIntegrationTestHelper) staticCallEVM(
	ctx *plugin.FakeContextWithEVM, method string, result interface{}, params ...interface{},
) error {
	input, err := c.contractABI.Pack(method, params...)
	if err != nil {
		return err
	}
	vm := evm.NewLoomVm(ctx.State, nil, nil, ctx.AccountBalanceManager, nil, false)
	output, err := vm.StaticCall(ctx.Message().Sender, c.Address, input)
	if err != nil {
		return err
	}
	return c.contractABI.Unpack(result, method, output)
}

func deployContractToEVM(ctx *plugin.FakeContextWithEVM, filename string, caller loom.Address) (loom.Address, error) {
	contractAddr := loom.Address{}
	hexByteCode, err := ioutil.ReadFile("testdata/" + filename + ".bin")
	if err != nil {
		return contractAddr, err
	}
	byteCode := common.FromHex(string(hexByteCode))

	vm := evm.NewLoomVm(ctx.State, nil, nil, nil, nil, false)
	_, contractAddr, err = vm.Create(caller, byteCode, loom.NewBigUIntFromInt(0))
	if err != nil {
		return contractAddr, err
	}

	ctx.RegisterContract("", contractAddr, caller)
	return contractAddr, nil
}

func sciNot(m, n int64) *big.Int {
	ret := big.NewInt(10)
	ret.Exp(ret, big.NewInt(n), nil)
	ret.Mul(ret, big.NewInt(m))
	return ret
}
