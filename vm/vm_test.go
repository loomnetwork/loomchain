package vm

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"

	"loom"
	"loom/store"
	"io/ioutil"
	"fmt"
	"os"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"bytes"
	"encoding/binary"
)

func mockState() loom.State {
	header := abci.Header{}
	return loom.NewStoreState(context.Background(), store.NewMemStore(), header)
}

// Pseudo code
// loomToken = new LoomToken()
// delegateCallToken = new DelegateCallToken()
// transferGateway = new (loomToken, delegateCallToken, 0)
// loomToken.transfer( transferGateway, 10)  -> returns true
func TestProcessDeployTx(t *testing.T) {
	//Create state database
	evmStore := evmState {
		mockState(),
		*new(state.StateDB),
	}
	evmStore.evmDB = getStatDB(t)

	addrLoomToken := createLoomToken(t, evmStore)
	addrDelegateCallToken := createDelegateCallToken(t, evmStore)
	addrTransferGateway := createTransferGateway(t, evmStore, addrLoomToken, addrDelegateCallToken)
	_ = callTransfer(t, evmStore, addrLoomToken, addrTransferGateway, uint64(10))
}

func getStatDB(t *testing.T) state.StateDB {
	bytecode := common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00")
	_, sdb, err := runtime.Execute(bytecode, nil, nil)
	require.Nil(t, err)
	return *sdb
}

func createLoomToken(t *testing.T, evmStore evmState) ([]byte) {
	var res loom.TxHandlerResult
	loomTokenData := getContractData("./testdata/LoomToken.json")
	loomTokenTx := &DeployTx{
		To: &loom.Address{
			ChainId: "mock",
			Local:   []byte{},
		},
		Code: common.Hex2Bytes(snipOx(loomTokenData.Bytecode)),
	}
	loomTokenB, err := proto.Marshal(loomTokenTx)
	require.Nil(t, err)

	res, err = ProcessDeployTx(evmStore, loomTokenB)

	require.Nil(t, err)
	result := res.Tags[0].Value
	if !checkEqual(result, common.Hex2Bytes(snipOx(loomTokenData.DeployedBytecode))) {
		t.Error("create loomToken did not return deployed bytecode")
	}
	return res.Tags[1].Value
}

func createDelegateCallToken(t *testing.T, evmStore evmState) ([]byte) {
	var res loom.TxHandlerResult
	delegateCallTokenData := getContractData("./testdata/DelegateCallToken.json")
	delegateCallTokenTx := &DeployTx{
		To: &loom.Address{
			ChainId: "mock",
			Local:   []byte{},
		},
		Code: common.Hex2Bytes(snipOx(delegateCallTokenData.Bytecode)),
	}
	delegateCallTokenB, err := proto.Marshal(delegateCallTokenTx)
	require.Nil(t, err)

	res, err = ProcessDeployTx(evmStore, delegateCallTokenB)

	require.Nil(t, err)
	result := res.Tags[0].Value
	if !checkEqual(result, common.Hex2Bytes(snipOx(delegateCallTokenData.DeployedBytecode))) {
		t.Error("create delegate call token did not return deployed bytecode")
	}
	return res.Tags[1].Value
}

func createTransferGateway(t *testing.T, evmStore evmState, loomAdr, delAdr []byte) ([]byte) {
	var empty []byte
	var res loom.TxHandlerResult
	transferGatewayData := getContractData("./testdata/TransferGateway.json")
	inParams := evmParamsB(common.Hex2Bytes(snipOx(transferGatewayData.Bytecode)), loomAdr, delAdr, empty)
	transferGatewayTx := &DeployTx{
		To: &loom.Address{
			ChainId: "mock",
			Local:   []byte{},
		},
		Code: inParams,
	}
	transferGatewayB, err := proto.Marshal(transferGatewayTx)
	require.Nil(t, err)

	res, err = ProcessDeployTx(evmStore, transferGatewayB)
	require.Nil(t, err)
	result := res.Tags[0].Value
	if !checkEqual(result, common.Hex2Bytes(snipOx(transferGatewayData.DeployedBytecode))) {
		t.Error("create transfer Gateway did not return deployed bytecode")
	}
	return res.Tags[1].Value
}

func callTransfer(t *testing.T, evmStore evmState, addr1, addr2 []byte, amount uint64) (bool) {
	var res loom.TxHandlerResult
	inParams := evmParams("transfer(address,uint256)", addr2,  uint64ToByte(amount))
	transferTx := &DeployTx{
		To: &loom.Address{
			ChainId: "mock",
			Local:   addr1,
		},
		Code: inParams,
	}
	transferTxB, err := proto.Marshal(transferTx)
	require.Nil(t, err)

	res, err = ProcessSendTx(evmStore, transferTxB)
	result := res.Tags[0].Value
	if (checkEqual(result, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1})) {
		return true
	} else if (checkEqual(result, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0})) {
		return false
	} else {
		t.Error("calling transfer did not return boolean")
	}
	return false
}

func getContractData(filename string) (ContractData) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var data ContractData
	err2 := json.Unmarshal(raw, &data)
	if (err2 != nil) {
		fmt.Println("Unmarshall error", err2)
	}
	return data
}
type ContractData struct {
	ContractName    string 				`json:"contractName"`
	Abi []interface{}					`json:"abi"`
	Bytecode   string					`json:"bytecode"`
	DeployedBytecode string				`json:"deployedBytecode"`
	SourceMap string					`json:"sourceMap"`
	DeployedSourceMap string			`json:"deployedSourceMap"`
	Source string						`json:"source"`
	SourcePath string					`json:"sourcePath"`
	Ast map[string]interface{}			`json:"ast"`
	LegacyAST map[string]interface{}	`json:"legacyAST"`
	Compiler map[string]interface{}		`json:"compiler"`
	Networks map[string]interface{}		`json:"networks"`
	SchemaVersion string				`json:"schemaVersion"`
	UpdateAI string						`json:"updatedAt"`
}

func snipOx(hexString string) (string) {
	if hexString[0:2] == string("0x") {
		hexString = hexString[2:]
	}
	return hexString
}

func checkEqual(b1, b2 []byte) bool {
	if b1 == nil && b2 == nil {
		return true;
	}
	if b1 == nil || b2 == nil {
		return false;
	}
	if len(b1) != len(b2) {
		return false
	}
	for i := range b1 {
		if b1[i] != b2[i] {
			return false
		}
	}
	return true
}

func evmParams(funcName string, params ...[]byte) ([]byte){
	funcNameToBytes := []byte(funcName)
	d := sha3.NewKeccak256()
	d.Write(funcNameToBytes)
	hashedFuncName := d.Sum(nil)
	clipedHashedFuncName := hashedFuncName[0:4]
	return evmParamsB(clipedHashedFuncName, params...)
}

func evmParamsB(prefix []byte, params ...[]byte) ([]byte){
	padded := prefix
	for _, param := range params {
		paramPadded := common.LeftPadBytes(param, 32)
		padded = append(padded, paramPadded...)
	}
	return padded
}

func uint64ToByte(theInt uint64) ([]byte) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, theInt)
	if err != nil {
		fmt.Println("error coverting int to bytes", err)
	}
	return buf.Bytes()
}