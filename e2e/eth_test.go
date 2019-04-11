package main

import (
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
		{"blockNumber", "eth-1-test.toml", 4, 10, "empty-genesis.json", "eth-test-loom.yaml"},
	}
	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators, test.accounts, 0)
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
