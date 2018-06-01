package main

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain/vm"
)

var (
	sSAddrChainId string
	sSAddrLocal   loom.LocalAddress
	tempDir       string
	logBuf        bytes.Buffer

	priFile = "pri"
	pubFile = "pub"

	// Need to wait for the new loom node to start up before continuing tests.
	sleepTime = 7 * time.Second

	// Input data used for deploy and call tests. This data is writin out to a file in the
	// temporty test directory before runing command
	// The test smart contract is the default SimpleStore from EthFiddle, https://ethfiddle.com/
	sSBytecode  = []byte("6060604052341561000f57600080fd5b60d38061001d6000396000f3006060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058202d9a0979adf6bf48461f24200e635bc19cd1786efbcfc0608eb1d76114d405860029")
	inputSet987 = []byte("60fe47b100000000000000000000000000000000000000000000000000000000000003db")
	inputGet    = []byte("6d4ce63c")

	tEventsBc = []byte("60606040523415600e57600080fd5b60ca8061001c6000396000f300606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063d0a2d2cb146044575b600080fd5b3415604e57600080fd5b606260048080359060200190919050506064565b005b7f6c2b4666ba8da5a95717621d879a77de725f3d816709b9cbe9f059b8f875e284816040518082815260200191505060405180910390a1505600a165627a7a7230582024693034eff3079e25dbd41c62b4396300f3eb77ab4bbf86910aa2f22866717e0029")
	tEventABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"i\",\"type\":\"uint256\"}],\"name\":\"sendEvent\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"number\",\"type\":\"uint256\"}],\"name\":\"MyEvent\",\"type\":\"event\"}]"

	// Data to check the return value of deploy and call transactions against.
	sSRuntimeBytecode = common.Hex2Bytes("6060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058202d9a0979adf6bf48461f24200e635bc19cd1786efbcfc0608eb1d76114d405860029")
	getReturnValue    = common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000003db")
	tEveentRuntimeBc  = common.Hex2Bytes("606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063d0a2d2cb146044575b600080fd5b3415604e57600080fd5b606260048080359060200190919050506064565b005b7f6c2b4666ba8da5a95717621d879a77de725f3d816709b9cbe9f059b8f875e284816040518082815260200191505060405180910390a1505600a165627a7a7230582024693034eff3079e25dbd41c62b4396300f3eb77ab4bbf86910aa2f22866717e0029")
)

func TestMain(m *testing.M) {
	var err error
	tempDir, err = ioutil.TempDir("", "testloom")
	if err != nil {
		panic("Error making temp directory")
	}
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	if err != nil {
		panic("Error seting working directory")
	}

	log.SetOutput(&logBuf)

	os.Exit(m.Run())
}

// ./loom init
func TestInit(t *testing.T) {
	var finit *cobra.Command
	finit = newInitCommand()
	err := finit.RunE(RootCmd, []string{})
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	genesisFile := filepath.Join(tempDir, "genesis.json")
	if _, err := os.Stat(genesisFile); err != nil {
		t.Fatalf("init should created a genesis.json file")
	}
}

// ./loom run
func TestRun(t *testing.T) {
	t.Skip("non isolated test")
	var frun *cobra.Command
	frun = newRunCommand()
	go frun.RunE(RootCmd, []string{})
	time.Sleep(sleepTime)

	appdb := filepath.Join(tempDir, "app.db")
	if _, err := os.Stat(appdb); err != nil {
		t.Fatalf("run should have created an appdb directory")
	}
}

// ./loom genkey -a "pub" -k "pri
// Results written to files. Used in TestDeploy and TestCall which helps to comfirms them.
func TestGenKey(t *testing.T) {
	var fgenkey *cobra.Command
	fgenkey = newGenKeyCommand()
	flags := fgenkey.Flags()
	flags.Set("public_key", pubFile)
	flags.Set("private_key", priFile)
	err := fgenkey.RunE(RootCmd, []string{})
	if err != nil {
		t.Fatalf("genkey returned error: %v", err)
	}
	if _, err := os.Stat(pubFile); err != nil {
		t.Fatalf("genkey did not make public key, error: %v", err)
	}
	if _, err := os.Stat(priFile); err != nil {
		t.Fatalf("genkey did not make private key, error: %v", err)
	}
}

// ./loom deploy -a <datapath>/pub -k <datapath>/pri -b <datapath>/simplestore.bin
// This deploloys the SimpleStore.sol smart contract from https://ethfiddle.com/ .
// Use deployTx rather than newDeployCommand().RunE to more easily access return values,
// the contract address returned here is used in the TestCall below.
func TestDeploy(t *testing.T) {
	t.Skip("non isolated test")
	bytefile := "simplestore.bin"
	err := ioutil.WriteFile(bytefile, sSBytecode, 0644)
	if err != nil {
		t.Fatalf("Error writing file, %v", err)
	}
	overrideChainFlags(chainFlags{
		ChainID:  "default",
		WriteURI: "http://localhost:46658/rpc",
		ReadURI:  "http://localhost:46658/query",
	})
	addr, runcode, _, err := deployTx(bytefile, priFile, pubFile, "")

	sSAddrChainId = addr.ChainID
	sSAddrLocal = addr.Local
	if err != nil {
		t.Fatalf("Error deploying contract %v", err)
	}
	if bytes.Compare(runcode, sSRuntimeBytecode) != 0 {
		t.Fatalf("Expected %s", sSRuntimeBytecode)
		t.Fatalf("Got %s", runcode)
		t.Fatalf("Wrong runtime bytecode returned")
	}
	if addr.IsEmpty() {
		t.Fatalf("Empty address returned")
	}
}

// ./loom deploy -a <datapath>/pub -k <datapath>/pri -b <datapath>/inputSet987.bin -c <contract addr>
// ./loom deploy -a <datapath>/pub -k <datapath>/pri -b <datapath>/inputGet.bin -c <contract addr>
// The SimpleStore contract has two members, set and get. Here we set the value of the store
// then use get and confirm we return the value we set it to.
// Use callTx rather than newCallCommand().RunE to more easily access return values,
func TestCall(t *testing.T) {
	t.Skip("non isolated test")
	sSAddr := loom.Address{
		ChainID: sSAddrChainId,
		Local:   sSAddrLocal,
	}
	set987file := "inputSet987.bin"
	err := ioutil.WriteFile(set987file, inputSet987, 0644)
	if err != nil {
		t.Fatalf("Error writing file, %v", err)
	}

	overrideChainFlags(chainFlags{
		ChainID:  "default",
		WriteURI: "http://localhost:46658/rpc",
		ReadURI:  "http://localhost:46658/query",
	})
	ret, err := callTx(sSAddr.String(), set987file, priFile, pubFile, "")
	if err != nil {
		t.Fatalf("Error on call set: %v", err)
	}
	if bytes.Compare(ret, nil) != 0 {
		t.Fatalf("Set should not return a value from set(987)")
	}

	inputGetFile := "inputGet.bin"
	err = ioutil.WriteFile(inputGetFile, inputGet, 0644)
	if err != nil {
		t.Fatalf("Error writing file, %v", err)
	}

	ret, err = callTx(sSAddr.String(), inputGetFile, priFile, pubFile, "")
	if err != nil {
		t.Fatalf("Error on call get: %v", err)
	}
	if bytes.Compare(ret, getReturnValue) != 0 {
		t.Fatalf("Expected %s", getReturnValue)
		t.Fatalf("Got %s", ret)
		t.Fatalf("Wrong value returned by get()")
	}
}

// This test deploys a contract with a sendEvent method that emits a simple event.
// We deploy the contract and call the sendEvent function.
// The in memory log is then parsed to find the output event
// and it is confirmed that the logged event contains the information from the
// ethereum log of the event.
func TestEvents(t *testing.T) {
	t.Skip("non isolated test")
	// Deploy the TestEvent contract
	err := ioutil.WriteFile("eventbf", tEventsBc, 0644)
	if err != nil {
		t.Fatalf("error writing to file: %v", err)
	}
	addr, runCode, _, err := deployTx("eventbf", priFile, pubFile, "")
	if err != nil {
		t.Fatalf("Error deploying TestEvent %v", err)
	}
	if bytes.Compare(runCode, tEveentRuntimeBc) != 0 {
		t.Fatalf("Expected %s", sSRuntimeBytecode)
		t.Fatalf("Got %s", runCode)
		t.Fatalf("Wrong runtime bytecode returned")
	}

	// Call the sendEvent method of TestEvent
	testNum := int64(76)
	abiEventData, err := abi.JSON(strings.NewReader(tEventABI))
	inParms, err := abiEventData.Pack("sendEvent", big.NewInt(testNum))
	err = ioutil.WriteFile("sendEventIn", []byte(common.Bytes2Hex(inParms)), 0644)
	if err != nil {
		t.Fatalf("Error writing file, %v", err)
	}
	_, err = callTx(addr.String(), "sendEventIn", priFile, pubFile, "")
	if err != nil {
		t.Fatalf("Error calling sendEvent, %v", err)
	}

	// Find the logged event in the in-memory log.
	// Unmarshal it into an Event structure.
	logString := string(logBuf.Bytes())
	logEvent, _, err := nextLoggedEvent(logString)
	if err != nil {
		t.Fatalf("error finding next event: %v", err)
	}
	event := vm.Event{}
	err = proto.Unmarshal([]byte(logEvent), &event)
	if err != nil {
		t.Fatalf("error unmarhalling event: %v", err)
	}
	eventAddr := loom.UnmarshalAddressPB(event.Contract)

	// Confirm that the event read in from the logger matches correctly the
	// solidity event it is derived from.
	// Address equals the contract address.
	// Topics equals a hash of the event signature, i.e. "MyEvent(uint256)"
	// Data equals the input uint256, in this case 76.
	if eventAddr.Compare(addr) != 0 {
		t.Fatalf("contract adderss %s does not match event adderess %s", addr.String(), eventAddr.String())
	}

	if len(event.Topics) != 1 {
		t.Fatalf("event should have only one topic, instead %d ", len(event.Topics))
	}
	d := sha3.NewKeccak256()
	d.Write([]byte("MyEvent(uint256)"))
	hashedEventSigniture := d.Sum(nil)
	if bytes.Compare(event.Topics[0], hashedEventSigniture) != 0 {
		t.Fatalf("topic does not match event signitre \"MyEvent(uint256)\"")
	}

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, testNum)
	if err != nil {
		t.Fatalf("error coverting int to bytes %v", err)
	}
	if bytes.Compare(event.Data, common.LeftPadBytes(buf.Bytes(), 32)) != 0 {
		t.Fatal("data does not match")
	}
}

func nextLoggedEvent(inLog string) (string, int, error) {
	eventId := strings.Index(inLog, "Event emitted:")
	if eventId < 0 {
		return "", eventId, nil
	}
	lenId := eventId + strings.Index(inLog[eventId:], "length: ")
	if eventId < 0 {
		return "", eventId, nil
	}
	msgId := lenId + strings.Index(inLog[lenId:], "msg: ")
	if eventId < 0 {
		return "", eventId, nil
	}
	start := msgId + len("msg: ")
	length, err := strconv.Atoi(inLog[lenId+len("length: ") : msgId-len(", ")])
	if err != nil {
		return "", start, err
	}
	return inLog[start : start+length], start + length, nil
}

// # Solidity contracts used in tests
//
// ## For test TestDeploy and TestCall
//
// pragma solidity ^0.4.18;
// contract SimpleStore {
//
//  function set(uint _value) public {
//      value = _value;
//  }
//
//  function get() public constant returns (uint) {
//      return value;
//  }
//
//  uint value;
// }
//
// ## For test TestEvents
//
// pragma solidity ^0.4.18;
// contract TestEvent {
//  event MyEvent(uint number);
//
//  function sendEvent(uint i) public   {
//      MyEvent(i);
//  }
// }
