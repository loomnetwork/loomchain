package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestDeployGoE2E(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
		yamlFile   string
	}{
		{"deployGo", "deploygo-1-test.toml", 4, 2, "empty-genesis.json", ""},
		{"whitelist", "deploygo-2-test.toml", 1, 3, "empty-genesis.json", "deploygo-2-loom.yaml"},
	}
	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators, test.accounts, 0)
			if err != nil {
				t.Fatal(err)
			}

			binary, err := exec.LookPath("go")
			if err != nil {
				t.Fatal(err)
			}

			cmdCli := exec.Cmd{
				Dir:  config.BaseDir,
				Path: binary,
				Args: []string{binary, "build", "-tags", "evm", "-o", "example-cli", "github.com/loomnetwork/go-loom/examples/cli"},
			}
			if err := cmdCli.Run(); err != nil {
				t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmdCli.Args, " "), err))
			}

			if err := common.DoRun(*config); err != nil {
				t.Fatal(err)
			}

			// pause before running the next test
			time.Sleep(500 * time.Millisecond)
		})
	}
}
