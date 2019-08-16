package main

import (
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int // TODO this is more like # of nodes than validators
		// # of validators is set in genesis params...
		accounts int
		genFile  string
		yamlFile string
	}{
		{
			"deployerwhitelist", "deployerwhitelist.toml", 2, 2,
			"deployerwhitelist.genesis.json", "deployerwhitelist-loom.yaml",
		},
		{
			"userdeployerwhitelist", "userdeployerwhitelist.toml", 1, 6,
			"userdeployerwhitelist.genesis.json", "userdeployerwhitelist-loom.yaml",
		},
		{
			"tx-limiter", "tx-limiter-test.toml", 1, 4, "", "tx-limiter-loom.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators,
				test.accounts, 0, false)
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
