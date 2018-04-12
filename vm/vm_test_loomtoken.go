package vm

import (
	"github.com/gogo/protobuf/proto"
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
func testLoomTokens(t *testing.T) {
	loomState := mockState()
	addrLoomToken := createToken(t, loomState, "./testdata/LoomToken.json")
	addrDelegateCallToken := createToken(t, loomState, "./testdata/DelegateCallToken.json")
	addrTransferGateway := createTransferGateway(t, loomState, addrLoomToken, addrDelegateCallToken)
	_ = callTransfer(t, loomState, addrLoomToken, addrTransferGateway, uint64(10))
}

func createToken(t *testing.T, loomState loom.State, filename string ) ([]byte) {
	var res loom.TxHandlerResult
	loomTokenData := getContractData(filename)
	loomTokenTx := &DeployTx{
		Input: common.Hex2Bytes(snipOx(loomTokenData.Bytecode)),
	}
	loomTokenB, err := proto.Marshal(loomTokenTx)
	require.Nil(t, err)

	res, err = ProcessDeployTx(loomState, loomTokenB)

	require.Nil(t, err)
	result := res.Tags[0].Value
	if !checkEqual(result, common.Hex2Bytes(snipOx(loomTokenData.DeployedBytecode))) {
		t.Error("create did not return deployed bytecode")
	}
	return res.Tags[1].Value
}

func createTransferGateway(t *testing.T, loomState loom.State, loomAdr, delAdr []byte ) ([]byte) {
	var empty []byte
	var res loom.TxHandlerResult
	transferGatewayData := getContractData("./testdata/TransferGateway.json")
	inParams := evmParamsB(common.Hex2Bytes(snipOx(transferGatewayData.Bytecode)), loomAdr, delAdr, empty)
	transferGatewayTx := &DeployTx{
		Input: inParams,
	}
	transferGatewayB, err := proto.Marshal(transferGatewayTx)
	require.Nil(t, err)

	res, err = ProcessDeployTx(loomState, transferGatewayB)
	require.Nil(t, err)
	result := res.Tags[0].Value
	if !checkEqual(result, common.Hex2Bytes(snipOx(transferGatewayData.DeployedBytecode))) {
		t.Error("create transfer Gateway did not return deployed bytecode")
	}
	return res.Tags[1].Value
}

func callTransfer(t *testing.T, loomState loom.State, addr1, addr2 []byte, amount uint64) (bool) {
	var res loom.TxHandlerResult
	inParams := evmParams("transfer(address,uint256)", addr2,  uint64ToByte(amount))
	transferTx := &SendTx{
		Address: addr1,
		Input: inParams,
	}
	transferTxB, err := proto.Marshal(transferTx)
	require.Nil(t, err)

	res, err = ProcessSendTx(loomState, transferTxB)

	result := res.Tags[0].Value
	if (checkEqual(result, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1})) {
		return true
	} else if (checkEqual(result, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0})) {
		t.Error("calling transfer returned false instead of true")
		return false
	} else {
		t.Error("calling transfer did not return boolean")
	}
	return false
}



