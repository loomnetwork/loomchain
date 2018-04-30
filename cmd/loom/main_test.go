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
	testDataPath  string
	sSAddrChainId string
	tempDir       string
	sSAddrLocal   loom.LocalAddress

	// Need to wait for the new loom node to start up before continuing tests.
	sleepTime = 7 * time.Second

	sSRuntimeBytecode = common.Hex2Bytes("6060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058202d9a0979adf6bf48461f24200e635bc19cd1786efbcfc0608eb1d76114d405860029")

	// inputGet          = common.Hex2Bytes("6d4ce63c")
	// inputSet987       = common.Hex2Bytes("60fe47b100000000000000000000000000000000000000000000000000000000000003db")
	getReturnValue = common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000003db")
)

func TestMain(m *testing.M) {
	loomDir, err := os.Getwd()
	if err != nil {
		panic("Error finding working directory")
	}

	testDataPath = filepath.Join(loomDir, "testdata")
	if err != nil {
		panic("Error setting testdata path")
	}

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
	finit.RunE(RootCmd, []string{})

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

// ./loom deploy -a <datapath>/pub -k <datapath>/pri -b <datapath>/simplestore.bin
// This deploloys the SimpleStore.sol smart contract from https://ethfiddle.com/ .
// Use deployTx rather than newDeployCommand().RunE to more easily access return values,
// the contract address returned here is used in the TestCall below.
func TestDeploy(t *testing.T) {
	t.Skip("skip broken test for now")
	bytefile := filepath.Join(testDataPath, "simplestore.bin")
	pubfile := filepath.Join(testDataPath, "pub")
	prifile := filepath.Join(testDataPath, "pri")

	addr, runcode, err := deployTx(bytefile, prifile, pubfile)

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
	t.Skip("skip broken test for now")
	pubfile := filepath.Join(testDataPath, "pub")
	prifile := filepath.Join(testDataPath, "pri")
	sSAddr := loom.Address{
		ChainID: sSAddrChainId,
		Local:   sSAddrLocal,
	}
	set987file := filepath.Join(testDataPath, "inputSet987.bin")

	ret, err := callTx(sSAddr.String(), set987file, prifile, pubfile)
	if err != nil {
		t.Fatalf("Error on call set: %v", err)
	}
	if bytes.Compare(ret, nil) != 0 {
		t.Fatalf("Set should not return a value from set(987)")
	}

	getFile := filepath.Join(testDataPath, "inputGet.bin")

	ret, err = callTx(sSAddr.String(), getFile, prifile, pubfile)
	if err != nil {
		t.Fatalf("Error on call get: %v", err)
	}
	if bytes.Compare(ret, getReturnValue) != 0 {
		t.Fatalf("Expected %s", getReturnValue)
		t.Fatalf("Got %s", ret)
		t.Fatalf("Wrong value returned by get()")
	}
}
