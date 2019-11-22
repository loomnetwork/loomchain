// +build evm

package main

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/log"
)

func configureGeth(cfg *config.GethConfig) {
	log.Info(
		"go-ethereum cfg",
		"EnableStateObjectDirtyStorageKeysSorting", cfg.EnableStateObjectDirtyStorageKeysSorting,
		"EnableTrieDatabasePreimageKeysSorting", cfg.EnableTrieDatabasePreimageKeysSorting,
	)
	state.EnableStateObjectDirtyStorageKeysSorting = cfg.EnableStateObjectDirtyStorageKeysSorting
	trie.EnableTrieDatabasePreimageKeysSorting = cfg.EnableTrieDatabasePreimageKeysSorting
}
