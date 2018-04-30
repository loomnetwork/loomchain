package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
)

var (
	sSAddrChainId string
	sSAddrLocal   loom.LocalAddress
	tempDir       string

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

	// Data to check the return value of deploy and call transactions against.
	sSRuntimeBytecode = common.Hex2Bytes("6060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058202d9a0979adf6bf48461f24200e635bc19cd1786efbcfc0608eb1d76114d405860029")
	getReturnValue    = common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000003db")
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
	flags.Set("address", pubFile)
	flags.Set("key", priFile)
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
func TestDeploly(t *testing.T) {
	bytefile := "simplestore.bin"
	err := ioutil.WriteFile(bytefile, sSBytecode, 0644)
	if err != nil {
		t.Fatalf("Error writing file, %v", err)
	}
	addr, runcode, err := deployTx(bytefile, priFile, pubFile)

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
	sSAddr := loom.Address{
		ChainID: sSAddrChainId,
		Local:   sSAddrLocal,
	}
	set987file := "inputSet987.bin"
	err := ioutil.WriteFile(set987file, inputSet987, 0644)
	if err != nil {
		t.Fatalf("Error writing file, %v", err)
	}

	ret, err := callTx(sSAddr.String(), set987file, priFile, pubFile)
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

	ret, err = callTx(sSAddr.String(), inputGetFile, priFile, pubFile)
	if err != nil {
		t.Fatalf("Error on call get: %v", err)
	}
	if bytes.Compare(ret, getReturnValue) != 0 {
		t.Fatalf("Expected %s", getReturnValue)
		t.Fatalf("Got %s", ret)
		t.Fatalf("Wrong value returned by get()")
	}
}
