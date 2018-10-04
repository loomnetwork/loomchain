package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestContractCoin(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
		yamlFile   string
	}{
		{"coin-1", "coin.toml", 1, 10, "coin.genesis.json", ""},
		{"coin-2", "coin.toml", 2, 10, "coin.genesis.json", ""},
		{"coin-2-r2", "coin.toml", 2, 10, "coin.genesis.json", "loom-receipts-v2.yaml"},
		{"coin-4", "coin.toml", 4, 10, "coin.genesis.json", ""},
		{"coin-4-r2", "coin.toml", 4, 10, "coin.genesis.json", "loom-receipts-v2.yaml"},
		{"coin-6", "coin.toml", 6, 10, "coin.genesis.json", ""},
		{"coin-8", "coin.toml", 8, 10, "coin.genesis.json", ""},
	}
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
			// required binary
			cmd := exec.Cmd{
				Dir:  config.BaseDir,
				Path: binary,
				Args: []string{binary, "build", "-o", "example-cli", "github.com/loomnetwork/go-loom/examples/cli"},
			}
			if err := cmd.Run(); err != nil {
				t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmd.Args, " "), err))
			}

			if err := common.DoRun(*config); err != nil {
				t.Fatal(err)
			}
		})
	}
}
