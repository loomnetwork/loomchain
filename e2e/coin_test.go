package main

import (
	"testing"
	"time"

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
		{"coin-4", "coin.toml", 4, 10, "coin.genesis.json", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(
				test.name,
				test.testFile,
				test.genFile,
				test.yamlFile,
				test.validators,
				test.accounts,
				0,
				false,
			)
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
