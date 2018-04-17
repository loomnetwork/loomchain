package vm

import (
	"testing"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
	"math/big"
	"fmt"
)

func testCryptoZombies(t *testing.T, vm VM, caller loom.Address) {
	motherKat := loom.Address{
		ChainID: "AChainID",
		Local:  []byte("myMotherKat"),
	}

	kittyData := GetFiddleContractData("./testdata/KittyInterface.json")
	zAttackData := GetFiddleContractData("./testdata/ZombieAttack.json")
	zFactoryData := GetFiddleContractData("./testdata/ZombieFactory.json")
	zFeedingData := GetFiddleContractData("./testdata/ZombieFeeding.json")
	zHelperData := GetFiddleContractData("./testdata/ZombieHelper.json")
	zOwnershipData := GetFiddleContractData("./testdata/ZombieOwnership.json")
	kittyAddr := deployContract(t, vm, motherKat, kittyData.Bytecode, kittyData.RuntimeBytecode)
	deployContract(t, vm, caller, zAttackData.Bytecode, zAttackData.RuntimeBytecode)
	zFactoryAddr := deployContract(t, vm, caller, zFactoryData.Bytecode, zFactoryData.RuntimeBytecode)
	zFeedingAddr := deployContract(t, vm, caller, zFeedingData.Bytecode, zFeedingData.RuntimeBytecode)
	deployContract(t, vm, caller, zHelperData.Bytecode, zHelperData.RuntimeBytecode)
	deployContract(t, vm, caller, zOwnershipData.Bytecode, zOwnershipData.RuntimeBytecode)

	checkKitty(t, vm, caller, kittyAddr, kittyData)
	makeZombie(t, vm, caller, zFactoryAddr, zFactoryData, "EEK")
	hungryZombie := getZombies(t, vm, caller, zFactoryAddr, zFactoryData, 0)

	zombieFeed(t,vm, caller, zFeedingAddr, zFeedingData, 0, 67)
	fedZombie := getZombies(t, vm, caller, zFactoryAddr, zFactoryData, 0)
	if checkEqual(hungryZombie, fedZombie) {
		//t.Error("fed zombie should have different dna")
	}

}

func deployContract(t *testing.T, vm VM, caller loom.Address, code string, runCode string) (loom.Address) {
	res, addr, err := vm.Create(caller, common.Hex2Bytes(code))

	require.Nil(t, err)
	if !checkEqual(res, common.Hex2Bytes(runCode)) {
		t.Error("create did not return deployed bytecode")
	}
	return addr
}

func checkKitty(t *testing.T, vm VM, caller , contractAddr loom.Address, data FiddleContractData) ([]byte) {
	abiKitty, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read kitty interface ",err)
		return []byte{}
	}
	inParams, err := abiKitty.Pack("getKitty", big.NewInt(1))

	res, err := vm.StaticCall(caller, contractAddr, inParams)

	if !checkEqual(res, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 27, 80, 224, 91, 160, 181, 143}) {
		fmt.Println("getKitty should return (true, true, 3,4,5,6,7,8,9,7688748911342991) actually returned ",res)
		fmt.Println("7688748911342991 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x",7688748911342991)))
		t.Error("get kitty returned wrong value")
	}
	return res
}

func makeZombie(t *testing.T, vm VM, caller , contractAddr loom.Address,  data FiddleContractData, name string) ([]byte) {
	abiZFactory, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie factory interface ",err)
		return []byte{}
	}
	inParams, err := abiZFactory.Pack("createRandomZombie", name)
	require.Nil(t, err)
	res, err := vm.Call(caller, contractAddr, inParams)
	if (err != nil) {
		t.Error("Error on making zombie")
	}

	if !checkEqual(res, nil) {
		t.Error("create zombie should not return a value")
	} else {
		fmt.Println("Just made zombie for ", caller)
	}
	return res
}

func getZombies(t *testing.T, vm VM, caller , contractAddr loom.Address,  data FiddleContractData, id uint) ([]byte) {
	abiZFactory, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie factory interface ",err)
		return []byte{}
	}
	inParams, err := abiZFactory.Pack("zombies", big.NewInt(int64(id)))
	require.Nil(t, err)
	res, err := vm.Call(caller, contractAddr, inParams)
	if (err != nil) {
		t.Error("Error on making zombie")
	}
	//Returned
	//struct Zombie {
	//	string name;
	//	uint dna;
	//	uint32 level;
	//	uint32 readyTime;
	//	uint16 winCount;
	//	uint16 lossCount;
	//}
	fmt.Println("Zombie with id", id, " looks like ", res)
	return res
}

func zombieFeed(t *testing.T, vm VM, caller , contractAddr loom.Address, data FiddleContractData, zombieId, kittyId uint) ([]byte) {
	abiZHelper, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie helper interface ",err)
		return []byte{}
	}
	inParams, err := abiZHelper.Pack("feedOnKitty",big.NewInt(int64(zombieId)), big.NewInt(int64(kittyId)))
	require.Nil(t, err)
	res, err := vm.Call(caller, contractAddr, inParams)
	if !checkEqual(res, nil) {
		t.Error("feed on kitty should not return anything")
	} else {
		fmt.Println("fed zombie ", zombieId, " on kitty", kittyId)
	}
	return res
}
























