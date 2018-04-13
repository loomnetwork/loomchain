package vm

import (
	"github.com/stretchr/testify/require"
	"testing"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/common"
)

// Pseudo code
// loomToken = new LoomToken()
// delegateCallToken = new DelegateCallToken()
// transferGateway = new (loomToken, delegateCallToken, 0)
// loomToken.transfer( transferGateway, 10)  -> returns true
func testLoomTokens(t *testing.T, vm VM, caller loom.Address) {
	loomData := getContractData("./testdata/LoomToken.json")
	delegateCallData := getContractData("./testdata/DelegateCallToken.json")

	addrLoomToken := deployContract(t, vm, caller, snipOx(loomData.Bytecode), snipOx(loomData.DeployedBytecode))
	addrDelegateCallToken := deployContract(t, vm, caller, snipOx(delegateCallData.Bytecode), snipOx(delegateCallData.DeployedBytecode))
	addrTransferGateway := createTransferGateway(t, vm, caller, addrLoomToken, addrDelegateCallToken)
	_ = callTransfer(t, vm, caller, addrLoomToken, addrTransferGateway, uint64(10))
}

func createTransferGateway(t *testing.T, vm VM, caller, loomAdr, delAdr  loom.Address) (loom.Address) {
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

func callTransfer(t *testing.T, vm VM, caller, contractAddr, addr2  loom.Address, amount uint64) (bool) {
	inParams := evmParams("transfer(address,uint256)", addr2.Local,  uint64ToByte(amount))

	res, err := vm.Call(caller, contractAddr, inParams)

	require.Nil(t, err)
	if (checkEqual(res, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1})) {
		return true
	} else if (checkEqual(res, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0})) {
		t.Error("calling transfer returned false instead of true")
		return false
	} else {
		t.Error("calling transfer did not return boolean")
	}
	return false
}



