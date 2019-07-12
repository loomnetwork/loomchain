package main

import (
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestE2eKarma(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
		yamlFile   string
	}{
		{"karma", "karma-1-test.toml", 1, 3, "karma-1-test.json", "karma-1-loom.yaml"},
		{"coin", "karma-2-test.toml", 1, 4, "karma-2-test.json", "karma-2-loom.yaml"},
		{"upkeep", "karma-3-test.toml", 1, 4, "karma-3-test.json", "karma-3-loom.yaml"},
		{"config", "karma-4-test.toml", 1, 2, "karma-4-test.json", "karma-3-loom.yaml"},
	}
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
