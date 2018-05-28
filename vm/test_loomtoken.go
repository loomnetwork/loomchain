// +build evm

package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	lp "github.com/loomnetwork/go-loom"
)

// Pseudo code
// loomToken = new LoomToken()
// delegateCallToken = new DelegateCallToken()
// transferGateway = new (loomToken, delegateCallToken, 0)
// loomToken.transfer( transferGateway, 10)  -> returns true
func testLoomTokens(t *testing.T, vm VM, caller lp.Address) {
	loomData := getContractData("./testdata/LoomToken.json")
	delegateCallData := getContractData("./testdata/DelegateCallToken.json")

	addrLoomToken := deployContract(t, vm, caller, snipOx(loomData.Bytecode), snipOx(loomData.DeployedBytecode))
	addrDelegateCallToken := deployContract(t, vm, caller, snipOx(delegateCallData.Bytecode), snipOx(delegateCallData.DeployedBytecode))
	addrTransferGateway := createTransferGateway(t, vm, caller, addrLoomToken, addrDelegateCallToken)
	_ = callTransfer(t, vm, caller, addrLoomToken, addrTransferGateway, uint64(10))
}

func createTransferGateway(t *testing.T, vm VM, caller, loomAdr, delAdr lp.Address) lp.Address {
	var empty []byte
	transferGatewayData := getContractData("./testdata/TransferGateway.json")
	inParams := evmParamsB(common.Hex2Bytes(snipOx(transferGatewayData.Bytecode)), loomAdr.Local, delAdr.Local, empty)

	res, addr, err := vm.Create(caller, inParams)

	require.Nil(t, err)
	if !checkEqual(res, common.Hex2Bytes(snipOx(transferGatewayData.DeployedBytecode))) {
		t.Error("create transfer Gateway did not return deployed bytecode")
	}
	return addr
}

func callTransfer(t *testing.T, vm VM, caller, contractAddr, addr2 lp.Address, amount uint64) bool {
	inParams := evmParams("transfer(address,uint256)", addr2.Local, uint64ToByte(amount))

	_, err := vm.Call(caller, contractAddr, inParams)

	require.Nil(t, err)
	return false
}
