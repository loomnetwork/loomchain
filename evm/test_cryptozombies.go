// +build evm

package evm

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
)

func testCryptoZombies(t *testing.T, vm lvm.VM, caller loom.Address) {
	motherKat := loom.Address{
		ChainID: "AChainID",
		Local:   []byte("myMotherKat"),
	}

	kittyData := GetFiddleContractData("./testdata/KittyInterface.json")
	zOwnershipData := GetFiddleContractData("./testdata/ZombieOwnership.json")

	kittyAddr := deployContract(t, vm, motherKat, kittyData.Bytecode, kittyData.RuntimeBytecode)
	zOwnershipAddr := deployContract(t, vm, caller, zOwnershipData.Bytecode, zOwnershipData.RuntimeBytecode)

	checkKitty(t, vm, caller, kittyAddr, kittyData)

	makeZombie(t, vm, caller, zOwnershipAddr, zOwnershipData, "EEK")
	greedyZombie := getZombies(t, vm, caller, zOwnershipAddr, zOwnershipData, 0)
	// greedy zombie should look like:
	//{
	//"0": "string: name EEK",
	//"1": "uint256: dna 2925635026906600",
	//"2": "uint32: level 1",
	//"3": "uint32: readyTime 1523984404",
	//"4": "uint16: winCount 0",
	//"5": "uint16: lossCount 0"
	//}
	if !checkEqual(greedyZombie[57:64], []byte{10, 100, 217, 124, 133, 109, 232}) {
		fmt.Println("dna 2925635026906600 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x", 2925635026906600)))
		fmt.Println("new zombie data: ", greedyZombie)
		t.Error("Wrong dna for greedy zombie")
	}

	setKittyAddress(t, vm, caller, kittyAddr, zOwnershipAddr, zOwnershipData)
	zombieFeed(t, vm, caller, zOwnershipAddr, zOwnershipData, 0, 67)

	newZombie := getZombies(t, vm, caller, zOwnershipAddr, zOwnershipData, 1)
	// New zombie should look like
	//{
	//"0": "string: name NoName",
	//"1": "uint256: dna 5307191969124799",
	//"2": "uint32: level 1",
	//"3": "uint32: readyTime 1523984521",
	//"4": "uint16: winCount 0",
	//"5": "uint16: lossCount 0"
	//}
	if !checkEqual(newZombie[57:64], []byte{18, 218, 220, 236, 19, 17, 191}) {
		fmt.Println("dna 5307191969124799 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x", 5307191969124799)))
		fmt.Println("new zombie data: ", newZombie)
		t.Error("Wrong dna for new zombie")
	}

}

func testCryptoZombiesUpdateState(t *testing.T, state loomchain.State, caller loom.Address) {
	motherKat := loom.Address{
		ChainID: "AChainID",
		Local:   []byte("myMotherKat"),
	}
	manager := lvm.NewManager()
	manager.Register(lvm.VMType_PLUGIN, LoomVmFactory)

	kittyData := GetFiddleContractData("./testdata/KittyInterface.json")
	zOwnershipData := GetFiddleContractData("./testdata/ZombieOwnership.json")

	vm, _ := manager.InitVM(lvm.VMType_PLUGIN, state)
	kittyAddr := deployContract(t, vm, motherKat, kittyData.Bytecode, kittyData.RuntimeBytecode)
	zOwnershipAddr := deployContract(t, vm, caller, zOwnershipData.Bytecode, zOwnershipData.RuntimeBytecode)
	checkKitty(t, vm, caller, kittyAddr, kittyData)
	makeZombie(t, vm, caller, zOwnershipAddr, zOwnershipData, "EEK")
	greedyZombie := getZombies(t, vm, caller, zOwnershipAddr, zOwnershipData, 0)
	// greedy zombie should look like:
	//{
	//"0": "string: name EEK",
	//"1": "uint256: dna 2925635026906600",
	//"2": "uint32: level 1",
	//"3": "uint32: readyTime 1523984404",
	//"4": "uint16: winCount 0",
	//"5": "uint16: lossCount 0"
	//}
	if !checkEqual(greedyZombie[57:64], []byte{10, 100, 217, 124, 133, 109, 232}) {
		fmt.Println("dna 2925635026906600 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x", 2925635026906600)))
		fmt.Println("new zombie data: ", greedyZombie)
		t.Error("Wrong dna for greedy zombie")
	}

	setKittyAddress(t, vm, caller, kittyAddr, zOwnershipAddr, zOwnershipData)
	zombieFeed(t, vm, caller, zOwnershipAddr, zOwnershipData, 0, 67)
	newZombie := getZombies(t, vm, caller, zOwnershipAddr, zOwnershipData, 1)
	// New zombie should look like
	//{
	//"0": "string: name NoName",
	//"1": "uint256: dna 5307191969124799",
	//"2": "uint32: level 1",
	//"3": "uint32: readyTime 1523984521",
	//"4": "uint16: winCount 0",
	//"5": "uint16: lossCount 0"
	//}
	if !checkEqual(newZombie[57:64], []byte{18, 218, 220, 236, 19, 17, 191}) {
		fmt.Println("dna 5307191969124799 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x", 5307191969124799)))
		fmt.Println("new zombie data: ", newZombie)
		t.Error("Wrong dna for new zombie")
	}

}

func deployContract(t *testing.T, vm lvm.VM, caller loom.Address, code string, runCode string) loom.Address {
	res, addr, err := vm.Create(caller, common.Hex2Bytes(code), loom.NewBigUIntFromInt(0))
	require.NoError(t, err, "calling vm.Create")

	output := lvm.DeployResponseData{}
	err = proto.Unmarshal(res, &output)
	require.NoError(t, err)
	if !checkEqual(output.Bytecode, common.Hex2Bytes(runCode)) {
		t.Error("create did not return deployed bytecode")
	}

	testGetCode(t, vm, addr, runCode)

	return addr
}

func testGetCode(t *testing.T, vm lvm.VM, addr loom.Address, expectedCode string) {
	actualCode, err := vm.GetCode(addr)
	require.NoError(t, err)
	if !checkEqual(actualCode, common.Hex2Bytes(expectedCode)) {
		t.Error("wrong runcode returned by GetCode")
	}
}

func checkKitty(t *testing.T, vm lvm.VM, caller, contractAddr loom.Address, data FiddleContractData) []byte {
	abiKitty, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read kitty interface ", err)
		return []byte{}
	}
	inParams, err := abiKitty.Pack("getKitty", big.NewInt(1))
	if err != nil {
		t.Error("Error in getKitty ", err)
	}
	res, err := vm.StaticCall(caller, contractAddr, inParams)
	if err != nil {
		t.Error("Error in call", err)
	}
	//nolint:lll
	if !checkEqual(res, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 27, 80, 224, 91, 160, 181, 143}) {
		fmt.Println("getKitty should return (true, true, 3,4,5,6,7,8,9,7688748911342991) actually returned ", res)
		fmt.Println("7688748911342991 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x", 7688748911342991)))
		t.Error("get kitty returned wrong value")
	}
	return res
}

func makeZombie(
	t *testing.T, vm lvm.VM, caller, contractAddr loom.Address, data FiddleContractData, name string,
) []byte {
	abiZFactory, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie factory interface ", err)
		return []byte{}
	}
	inParams, err := abiZFactory.Pack("createRandomZombie", name)
	require.Nil(t, err)
	res, err := vm.Call(caller, contractAddr, inParams, loom.NewBigUIntFromInt(0))
	if err != nil {
		t.Error("Error on making zombie")
	}

	return res
}

func getZombies(t *testing.T, vm lvm.VM, caller, contractAddr loom.Address, data FiddleContractData, id uint) []byte {
	abiZFactory, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie factory interface ", err)
		return []byte{}
	}
	inParams, err := abiZFactory.Pack("zombies", big.NewInt(int64(id)))
	require.Nil(t, err)
	res, err := vm.StaticCall(caller, contractAddr, inParams)
	if err != nil {
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
	return res
}

func zombieFeed(
	t *testing.T, vm lvm.VM, caller, contractAddr loom.Address, data FiddleContractData, zombieId, kittyId uint,
) []byte {
	abiZFeeding, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie feeding interface ", err)
		return []byte{}
	}
	inParams, err := abiZFeeding.Pack("feedOnKitty", big.NewInt(int64(zombieId)), big.NewInt(int64(kittyId)))
	require.Nil(t, err)
	res, err := vm.Call(caller, contractAddr, inParams, loom.NewBigUIntFromInt(0))
	require.Nil(t, err)
	return res
}

func setKittyAddress(
	t *testing.T, vm lvm.VM, caller, kittyAddr, contractAddr loom.Address, data FiddleContractData,
) []byte {
	abiZFeeding, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read zombie feeding interface ", err)
		return []byte{}
	}
	inParams, err := abiZFeeding.Pack("setKittyContractAddress", common.BytesToAddress(kittyAddr.Local))
	require.Nil(t, err)
	res, err := vm.Call(caller, contractAddr, inParams, loom.NewBigUIntFromInt(0))
	if err != nil {
		t.Error("Error on setting kitty address")
	}
	return res
}
