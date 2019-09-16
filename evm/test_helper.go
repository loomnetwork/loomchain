// +build evm

package evm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

func getContractData(filename string) ContractData {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var data ContractData
	err2 := json.Unmarshal(raw, &data)
	if err2 != nil {
		fmt.Println("Unmarshall error", err2)
	}
	return data
}

type ContractData struct {
	ContractName      string                 `json:"contractName"`
	Abi               []interface{}          `json:"abi"`
	Bytecode          string                 `json:"bytecode"`
	DeployedBytecode  string                 `json:"deployedBytecode"`
	SourceMap         string                 `json:"sourceMap"`
	DeployedSourceMap string                 `json:"deployedSourceMap"`
	Source            string                 `json:"source"`
	SourcePath        string                 `json:"sourcePath"`
	Ast               map[string]interface{} `json:"ast"`
	LegacyAST         map[string]interface{} `json:"legacyAST"`
	Compiler          map[string]interface{} `json:"compiler"`
	Networks          map[string]interface{} `json:"networks"`
	SchemaVersion     string                 `json:"schemaVersion"`
	UpdateAI          string                 `json:"updatedAt"`
}

func GetFiddleContractData(filename string) FiddleContractData {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var data FiddleContractData
	err2 := json.Unmarshal(raw, &data)
	if err2 != nil {
		fmt.Println("Unmarshall error", err2)
	}
	return data
}

type FiddleContractData struct {
	Assembly        map[string]interface{} `json:"assembly"`
	Bytecode        string                 `json:"bytecode"`
	FunctionHashes  map[string]interface{} `json:"functionHashes"`
	GasEstimates    map[string]interface{} `json:"gasEstimates"`
	Iterface        string                 `json:"interface"`
	Metadata        string                 `json:"metadata"`
	Opcodes         string                 `json:"opcodes"`
	RuntimeBytecode string                 `json:"runtimeBytecode"`
	Srcmap          string                 `json:"srcmap"`
	SrcmapRuntime   string                 `json:"srcmapRuntime"`
}

func snipOx(hexString string) string {
	if hexString[0:2] == string("0x") {
		hexString = hexString[2:]
	}
	return hexString
}

func checkEqual(b1, b2 []byte) bool {
	if b1 == nil && b2 == nil {
		return true
	}
	if b1 == nil || b2 == nil {
		return false
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

func evmParams(funcName string, params ...[]byte) []byte {
	funcNameToBytes := []byte(funcName)
	d := sha3.NewLegacyKeccak256()
	d.Write(funcNameToBytes)
	hashedFuncName := d.Sum(nil)
	clipedHashedFuncName := hashedFuncName[0:4]
	return evmParamsB(clipedHashedFuncName, params...)
}

func evmParamsB(prefix []byte, params ...[]byte) []byte {
	padded := prefix
	for _, param := range params {
		paramPadded := common.LeftPadBytes(param, 32)
		padded = append(padded, paramPadded...)
	}
	return padded
}

func uint64ToByte(theInt uint64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, theInt)
	if err != nil {
		fmt.Println("error coverting int to bytes", err)
	}
	return buf.Bytes()
}