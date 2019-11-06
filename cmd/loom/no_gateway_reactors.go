// +build !gateway

package main

import (
	glAuth "github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/fnConsensus"
)

func startGatewayReactors(
	chainID string,
	fnRegistry fnConsensus.FnRegistry,
	cfg *config.Config,
	nodeSigner glAuth.Signer,
	app *loomchain.Application,
) error {
	return nil
}
