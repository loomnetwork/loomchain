package vm

import (
	"testing"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/gogo/protobuf/proto"
	"fmt"
	"strings"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
)

func testCryptoZombies(t *testing.T) {
	loomState := mockState()
	kittyData := GetFiddleContractData("./testdata/KittyInterface.json")
	zAttackData := GetFiddleContractData("./testdata/ZombieAttack.json")
	zFactoryData := GetFiddleContractData("./testdata/ZombieFactory.json")
	zFeedingData := GetFiddleContractData("./testdata/ZombieFeeding.json")
	zHelperData := GetFiddleContractData("./testdata/ZombieHelper.json")
	zOwnershipData := GetFiddleContractData("./testdata/ZombieOwnership.json")
	kittyAddr := deployContract(t, loomState, kittyData.Bytecode, kittyData.RuntimeBytecode)
	deployContract(t, loomState, zAttackData.Bytecode, zAttackData.RuntimeBytecode)
	zFactoryAddr := deployContract(t, loomState, zFactoryData.Bytecode, zFactoryData.RuntimeBytecode)
	deployContract(t, loomState, zFeedingData.Bytecode, zFeedingData.RuntimeBytecode)
	deployContract(t, loomState, zHelperData.Bytecode, zHelperData.RuntimeBytecode)
	deployContract(t, loomState, zOwnershipData.Bytecode, zOwnershipData.RuntimeBytecode)

	checkKitty(t, loomState, kittyAddr, kittyData)
	makeZombie(t, loomState, zFactoryAddr, zFactoryData, "EEK")

}

func deployContract(t *testing.T, loomState loom.State, code string, runCode string) ([]byte) {
	var res loom.TxHandlerResult
	loomTokenTx := &DeployTx{
		Input: common.Hex2Bytes(code),
	}
	loomTokenB, err := proto.Marshal(loomTokenTx)
	require.Nil(t, err)

	res, err = ProcessDeployTx(loomState, loomTokenB)

	require.Nil(t, err)
	result := res.Tags[0].Value
	if !checkEqual(result, common.Hex2Bytes(runCode)) {
		t.Error("create did not return deployed bytecode")
	}
	return res.Tags[1].Value
}

func checkKitty(t *testing.T, loomState loom.State, contractAddr []byte, data FiddleContractData) ([]byte) {
	var res loom.TxHandlerResult
	abiKitty, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read kitty interface ",err)
		return []byte{}
	}
	inParams, err := abiKitty.Pack("getKitty", big.NewInt(1))
	transferTx := &SendTx{
		Address: contractAddr,
		Input: inParams,
	}
	transferTxB, err := proto.Marshal(transferTx)
	require.Nil(t, err)

	res, err = ProcessSendTx(loomState, transferTxB)
	result := UnmarshalTxReturn(res)
	if !checkEqual(result, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 27, 80, 224, 91, 160, 181, 143}) {
		fmt.Println("getKitty should return (true, true, 3,4,5,6,7,8,9,7688748911342991) actually returned ",result)
		fmt.Println("7688748911342991 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x",7688748911342991)))
		t.Error("get kitty returned wrong value")
	}
	return result
}

func makeZombie(t *testing.T, loomState loom.State, contractAddr []byte, data FiddleContractData, name string) ([]byte) {
	var res loom.TxHandlerResult
	abiZFactory, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie factory interface ",err)
		return []byte{}
	}
	inParams, err := abiZFactory.Pack("createRandomZombie", name)
	transferTx := &SendTx{
		Address: contractAddr,
		Input: inParams,
	}
	transferTxB, err := proto.Marshal(transferTx)
	require.Nil(t, err)

	res, err = ProcessSendTx(loomState, transferTxB)
	result := UnmarshalTxReturn(res)
	if !checkEqual(result, nil) {
		t.Error("create zombie should not return a value")
	}
	return result
}



























