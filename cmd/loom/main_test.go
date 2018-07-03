package main

import (
	"bytes"
	"testing"
	"time"

	`github.com/loomnetwork/loomchain/e2e/common`
	`os/exec`
	`fmt`
	`strings`
)

var (
	tempDir       string
	logBuf        bytes.Buffer

	priFile = "pri"
	pubFile = "pub"

	// Need to wait for the new loom node to start up before continuing tests.
	sleepTime = 7 * time.Second
)


func TestE2eEvm(t *testing.T) {
	tests := []struct {
		testFile string
		n        int
		genFile  string
	}{
		{"evm-test.toml", 1, ""},
	}
	common.LoomPath = "../../loom"
	common.ContractDir = "../../contracts"
	for _, test := range tests {
		*common.Validators = test.n
		config, err := common.NewConfig("evm", test.testFile, test.genFile)
		if err != nil {
			t.Fatal(err)
		}
		
		binary, err := exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
		// required binary
		cmd := exec.Cmd{
			Dir:  config.BaseDir,
			Path: binary,
			Args: []string{
				binary,
				"build",
				"-tags",
				"evm",
				"-o",
				"loom",
				"github.com/loomnetwork/loomchain/cmd/loom",
			},
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmd.Args, " "), err))
		}
		
		if err := common.DoRun(*config); err != nil {
			t.Fatal(err)
		}
		
		// pause before running the next test
		time.Sleep(500 * time.Millisecond)
	}
}
/*
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

 */