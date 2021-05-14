package main

import (
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

type Test struct {
	name       string
	testFile   string
	validators int // TODO this is more like # of nodes than validators
	// # of validators is set in genesis params...
	accounts int
	genFile  string
	yamlFile string
}

func TestContractChainConfig(t *testing.T) {
	tests := []Test{
		/*
			{
				"chainconfig",
				"chainconfig.toml",
				4,
				4,
				"chainconfig.genesis.json",
				"chainconfig-loom.yaml",
			},
			{
				"enable-receipts-v2-feature",
				"enable-receipts-v2-feature.toml",
				1,
				1,
				"enable-receipts-v2-feature-genesis.json",
				"enable-receipts-v2-feature-loom.yaml",
			},
			{
				"chainconfig-routine",
				"chainconfig-routine.toml",
				4,
				4,
				"chainconfig.genesis.json",
				"chainconfig-routine-loom.yaml",
			},
		*/
		{
			"app-db-switchover",
			"app-db-switchover.toml",
			4,
			4,
			"app-db-switchover.genesis.json",
			"app-db-switchover.yaml",
		},
	}
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
