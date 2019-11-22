// +build evm

package main

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/loomnetwork/loomchain/config"
)

func configureGeth(cfg *config.GethConfig) {
	state.EnableStateObjectDirtyStorageKeysSorting = cfg.EnableStateObjectDirtyStorageKeysSorting
	trie.EnableTrieDatabasePreimageKeysSorting = cfg.EnableTrieDatabasePreimageKeysSorting
}
