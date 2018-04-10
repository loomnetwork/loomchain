package vm

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
	"io/ioutil"
	"fmt"
	"os"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
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