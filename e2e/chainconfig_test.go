package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestContractChainConfig(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int // TODO this is more like # of nodes than validators
		// # of validators is set in genesis params...
		accounts int
		genFile  string
		yamlFile string
	}{
		{"chainconfig", "chainconfig.toml", 4, 4, "chainconfig.genesis.json", "chainconfig-loom.yaml"},
		{"enable-receipts-v2-feature", "enable-receipts-v2-feature.toml", 1, 1, "enable-receipts-v2-feature-genesis.json", "enable-receipts-v2-feature-loom.yaml"},
	}

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

			cmd := exec.Cmd{
				Dir:  config.BaseDir,
				Path: binary,
				Args: []string{binary, "build", "-tags", "evm", "-o", "example-cli", "github.com/loomnetwork/go-loom/examples/cli"},
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
