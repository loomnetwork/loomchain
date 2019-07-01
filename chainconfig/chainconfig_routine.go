package chainconfig

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	goloom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/abci/backend"
	"github.com/loomnetwork/loomchain/config"
)

// ChainConfigRoutine periodically checks for pending features in the ChainConfig contract and
// automatically votes to enable those features.
type ChainConfigRoutine struct {
	cfg         *config.ChainConfigConfig
	chainID     string
	signer      auth.Signer
	address     goloom.Address
	logger      *goloom.Logger
	buildNumber uint64
	node        backend.Backend
}

// NewChainConfigRoutine returns a new instance of ChainConfigRoutine
func NewChainConfigRoutine(
	cfg *config.ChainConfigConfig,
	chainID string,
	nodeSigner auth.Signer,
	node backend.Backend,
	logger *goloom.Logger,
) (*ChainConfigRoutine, error) {
	address := goloom.Address{
		ChainID: chainID,
		Local:   goloom.LocalAddressFromPublicKey(nodeSigner.PublicKey()),
	}
	build, err := strconv.ParseUint(loomchain.Build, 10, 64)
	if err != nil {
		build = 0
	}
	return &ChainConfigRoutine{
		cfg:         cfg,
		chainID:     chainID,
		signer:      nodeSigner,
		address:     address,
		logger:      logger,
		buildNumber: build,
		node:        node,
	}, nil
}

// RunWithRecovery should be run as a go-routine, it will auto-restart on panic unless it hits
// a runtime error.
func (cc *ChainConfigRoutine) RunWithRecovery() {
	defer func() {
		if r := recover(); r != nil {
			cc.logger.Error("recovered from panic in ChainConfigRoutine", "r", r)
			// Unless it's a runtime error restart the goroutine
			if _, ok := r.(runtime.Error); !ok {
				time.Sleep(30 * time.Second)
				cc.logger.Info("Restarting ChainConfigRoutine.")
				go cc.RunWithRecovery()
			}
		}
	}()

	// Give the node a bit of time to spin up
	if cc.cfg.EnableFeatureStartupDelay > 0 {
		time.Sleep(time.Duration(cc.cfg.EnableFeatureStartupDelay) * time.Second)
	}
	cc.run()
}

func (cc *ChainConfigRoutine) run() {
	isBuildnumberEqual := false
	for {
		if cc.node.IsValidator() {
			dappClient := client.NewDAppChainRPCClient(cc.chainID, cc.cfg.DAppChainWriteURI, cc.cfg.DAppChainReadURI)
			chainConfigClient, err := NewChainConfigClient(dappClient, cc.address, cc.signer, cc.logger)
			if err != nil {
				cc.logger.Error("Failed to create ChainConfigClient", "err", err)
			} else {
				// NOTE: errors are logged by the client, no need to log again
				chainConfigClient.VoteToEnablePendingFeatures(cc.buildNumber)
				fmt.Println(isBuildnumberEqual)
				if !isBuildnumberEqual {
					validatorInfo, err := chainConfigClient.GetBuildNumber()
					if err != nil {
						cc.logger.Error("Failed to retreive build number", "err", err)
					}
					buildNumber := validatorInfo.Validator.GetBuildNumber()
					if buildNumber < cc.buildNumber {
						if err := chainConfigClient.SetBuildNumber(cc.buildNumber); err != nil {
							cc.logger.Error("Failed to set build number", "err", err)
						} else {
							isBuildnumberEqual = true
						}
					} else {
						isBuildnumberEqual = true
					}
				}
			}
		}
		time.Sleep(time.Duration(cc.cfg.EnableFeatureInterval) * time.Second)
	}
}
