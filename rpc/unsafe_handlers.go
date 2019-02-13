package rpc

//NOTE THIS PACKAGE IS TO  STRESS A TESTNET INSTANCE
//NEVER USE THIS CODE IN A REAL NETWORK

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/vm"
	"golang.org/x/crypto/ed25519"
)

type unsafeHandler struct {
	app *loomchain.Application
	svc QueryService
}

func newUnsafeHandler(app *loomchain.Application, svc QueryService) *unsafeHandler {
	return &unsafeHandler{app: app, svc: svc}
}

func (u *unsafeHandler) unsafeLoadDeliverTx(w http.ResponseWriter, req *http.Request) {
	//TODO for now we will always do a cpu profile
	//TODO read query string if, we need cpu, mem, or trace
	f, err := os.Create("cpu_profile_load_deliver_tx.txt")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	w.Write([]byte("unsafeLoadDeliverTx starting\n"))
	unsafeDeployEVMTestApp(u.app)

	//TODO read query string to know how many iteration
	to := loom.Address{}
	for i := 1; i < 100; i++ {
		unsafeLoadDeliverTx(u.app, i, to)
	}
	u.app.Commit()
	w.Write([]byte("unsafeLoadDeliverTx finished\n"))
	w.WriteHeader(200)
}

func unsafeDeployEVMTestApp(app *loomchain.Application) loom.Address {
	//TODO Deploy EVM TX
	contract := loom.MustParseAddress("default:0x9a1aC42a17AAD6Dbc6d21c162989d0f701074044")

	return contract
}

func unsafeLoadDeliverTx(app *loomchain.Application, round int, to loom.Address) {

	origBytes := []byte("origin")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	panicErr(err)

	origin := loom.Address{
		ChainID: "default",
		Local:   loom.LocalAddressFromPublicKey(pubKey),
	}

	var messageTx []byte

	deployTX, err := proto.Marshal(&vm.DeployTx{
		VmType: vm.VMType_EVM,
		Code:   origBytes,
	})
	panicErr(err)

	messageTx, err = proto.Marshal(&vm.MessageTx{
		Data: deployTX,
		To:   to.MarshalPB(),
		From: origin.MarshalPB(),
	})

	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   uint32(1),
		Data: messageTx,
	})
	nonceTx, err := proto.Marshal(&auth.NonceTx{
		Inner:    tx,
		Sequence: uint64(1), // uint64(round),
	})

	signer := auth.NewEd25519Signer(privKey)
	signedTx := auth.SignTx(signer, nonceTx)
	signedTxBytes, err := proto.Marshal(signedTx)
	panicErr(err)

	//TODO calling delivertx direct, we need to call via the tendermint api
	app.DeliverTx(signedTxBytes)
}

func panicErr(err error) {
	if err != nil {
		panic("Failed doing something:" + err.Error())
	}
}
func (u *unsafeHandler) unsafeTestCryptoZombiesHandler(w http.ResponseWriter, req *http.Request) {

}

func (u *unsafeHandler) unsafeTestCryptoZombies(caller loom.Address) {
	motherKat := loom.Address{
		ChainID: "AChainID",
		Local:   []byte("myMotherKat"),
	}

	kittyData := evm.GetFiddleContractData("./testdata/KittyInterface.json")
	zOwnershipData :=  evm.GetFiddleContractData("./testdata/ZombieOwnership.json")

	kittyAddr := u.deployContract(motherKat, kittyData.Bytecode, kittyData.RuntimeBytecode)
	zOwnershipAddr := u.deployContract(caller, zOwnershipData.Bytecode, zOwnershipData.RuntimeBytecode)

	u.checkKitty(caller, kittyAddr, kittyData)

	u.makeZombie(caller, zOwnershipAddr, zOwnershipData, "EEK")
	greedyZombie := u.getZombies(caller, zOwnershipAddr, zOwnershipData, 0)
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

	u.setKittyAddress(t, vm, caller, kittyAddr, zOwnershipAddr, zOwnershipData)
	u.zombieFeed(t, vm, caller, zOwnershipAddr, zOwnershipData, 0, 67)

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


func (u *unsafeHandler) makeZombie(caller, contractAddr loom.Address, data evm.FiddleContractData, name string) []byte {
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

func (u *unsafeHandler) checkKitty(caller, contractAddr loom.Address, data evm.FiddleContractData) []byte {
	abiKitty, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read kitty interface ", err)
		return []byte{}
	}
	inParams, err := abiKitty.Pack("getKitty", big.NewInt(1))

	res, err := vm.StaticCall(caller, contractAddr, inParams)
	if !checkEqual(res, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 27, 80, 224, 91, 160, 181, 143}) {
		fmt.Println("getKitty should return (true, true, 3,4,5,6,7,8,9,7688748911342991) actually returned ", res)
		fmt.Println("7688748911342991 as []byte is", common.Hex2Bytes(fmt.Sprintf("%x", 7688748911342991)))
		t.Error("get kitty returned wrong value")
	}
	return res
}

///------- duplicated code from the test base

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

func (u *unsafeHandler) deployContract(caller loom.Address, code string, runCode string) loom.Address {
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

func (u *unsafeHandler) testGetCode(vm lvm.VM, addr loom.Address, expectedCode string) {
	actualCode, err := vm.GetCode(addr)
	require.NoError(t, err)
	if !checkEqual(actualCode, common.Hex2Bytes(expectedCode)) {
		t.Error("wrong runcode returned by GetCode")
	}
}