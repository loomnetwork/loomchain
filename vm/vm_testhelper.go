package vm

import (
	"io/ioutil"
	abci "github.com/tendermint/abci/types"
	"context"
	"fmt"
	"os"
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/common"
	"bytes"
	"encoding/binary"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
)

func mockState() loom.State {
	header := abci.Header{}
	return loom.NewStoreState(
		context.Background(),
		store.NewMemStore(),
		header,
	)
}

/*
func mocSignedState(t *testing.T) loom.State {
	origBytes := []byte("hello")
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	signer := auth.NewEd25519Signer(privKey)
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	state := loom.NewStoreState(context.Background(), store.NewMemStore(), abci.Header{})
	auth.SignatureTxMiddleware.ProcessTx(state, signedTxBytes,
		func(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loom.TxHandlerResult{}, nil
		},
	)
	sender := &loom.Address{
		ChainID: state.Block().ChainID,
		Local:   origBytes,
	}
	a := context.WithValue(state.Context(), "authsender", &sender)
	a=a
	state2 := loom.NewStoreState(a, store.NewMemStore(), abci.Header{})
	return state2
}
*/

func UnmarshalTxReturn(res loom.TxHandlerResult) ([]byte){
	return res.Tags[0].Value
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

func GetFiddleContractData(filename string) (FiddleContractData) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var data FiddleContractData
	err2 := json.Unmarshal(raw, &data)
	if (err2 != nil) {
		fmt.Println("Unmarshall error", err2)
	}
	return data
}
type FiddleContractData struct {
	Assembly        map[string]interface{}    `json:"assembly"`
	Bytecode        string         `json:"bytecode"`
	FunctionHashes  map[string]interface{}   `json:"functionHashes"`
	GasEstimates    map[string]interface{}    `json:"gasEstimates"`
	Iterface        string			`json:"interface"`
	Metadata        string		   `json:"metadata"`
	Opcodes         string         `json:"opcodes"`
	RuntimeBytecode string         `json:"runtimeBytecode"`
	Srcmap          string         `json:"srcmap"`
	SrcmapRuntime   string         `json:"srcmapRuntime"`
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