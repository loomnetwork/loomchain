package main

import (
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestContractDPOS(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts int
		genFile  string
		yamlFile string
	}{
		// {"dpos-downtime", "dpos-downtime.toml", 4, 10, "dposv3.genesis.json", "dposv3-test-loom.yaml"}
		{"dpos-v3", "dposv3-delegation.toml", 4, 10, "dposv3.genesis.json", "dposv3-test-loom.yaml"},
		{"dpos-2", "dpos-2-validators.toml", 2, 10, "dposv3.genesis.json", "dposv3-test-loom.yaml"},
		{"dpos-2-r2", "dpos-2-validators.toml", 2, 10, "dposv3.genesis.json", "dposv3-test-loom.yaml"},
		{"dpos-4", "dpos-4-validators.toml", 4, 10, "dposv3-2.genesis.json", "dposv3-test-loom.yaml"},
		{"dpos-4-r2", "dpos-4-validators.toml", 4, 10, "dposv3-2.genesis.json", "dposv3-test-loom.yaml"},
		// {"dpos-8", "dpos-8-validators.toml", 8, 10, "dposv3-2.genesis.json", "dposv3-test-loom.yaml"},
		{"dpos-elect-time", "dpos-elect-time-2-validators.toml", 2, 10, "dpos-elect-time.genesis.json", "dpos-test-loom.yaml"},
	}

	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators, test.accounts, 0, false)
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
