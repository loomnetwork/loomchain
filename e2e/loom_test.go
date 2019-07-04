package main

import (
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestE2eEvm(t *testing.T) {
	tests := []struct {
		name        string
		testFile    string
		validators  int
		accounts    int
		ethAccounts int
		genFile     string
		yamlFile    string
	}{
		{"evm", "loom-1-test.toml", 4, 10, 0, "empty-genesis.json", "loom.yaml"},
		{"ethSignature-type1", "loom-3-test.toml", 1, 1, 1, "loom-3-genesis.json", "loom-3-loom.yaml"},
		{"ethSignature-type2", "loom-4-test.toml", 1, 2, 2, "loom-4-genesis.json", "loom-4-loom.yaml"},
		{"migration-tx", "loom-5-test.toml", 3, 3, 3, "loom-5-genesis.json", "loom-5-loom.yaml"},
		{"evm-state-migration", "loom-6-test.toml", 4, 4, 4, "loom-6-genesis.json", "loom-6-loom.yaml"},
		{"evm-name-option-not-allow", "loom-7-test.toml", 2, 2, 0, "loom-7-genesis.json", "loom-7-loom.yaml"},
		{"evm-name-option-allow", "loom-8-test.toml", 2, 2, 0, "loom-7-genesis.json", "loom-8-loom.yaml"},
	}
	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators, test.accounts, test.ethAccounts, false)
			if err != nil {
				t.Fatal(err)
			}

			if err := common.DoRun(*config); err != nil {
				t.Fatal(err)
			}

			// pause before running the next test
			time.Sleep(500 * time.Millisecond)
		})
	}
}
