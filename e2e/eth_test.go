package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestEthJSONRPC2(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
		yamlFile   string
	}{
		{"blockNumber", "eth-1-test.toml", 4, 0, "empty-genesis.json", "eth-loom.yaml"},
		{"getBlockByNumber", "eth-2-test.toml", 4, 1, "empty-genesis.json", "eth-loom.yaml"},
		{"getBlockTransactionCountByNumber", "eth-3-test.toml", 4, 1, "empty-genesis.json", "eth-loom.yaml"},
		{"getLogs", "eth-4-test.toml", 4, 4, "empty-genesis.json", "eth-loom.yaml"},
	}
	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators, test.accounts)
			if err != nil {
				t.Fatal(err)
			}

			binary, err := exec.LookPath("go")
			if err != nil {
				t.Fatal(err)
			}

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
		})
	}
}
