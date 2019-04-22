package chainconfig

import (
	"runtime"
	"strconv"
	"time"

	goloom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/config"
)

type ChainConfigRoutine struct {
	cfg         *config.ChainConfigConfig
	chainID     string
	signer      auth.Signer
	address     goloom.Address
	logger      *goloom.Logger
	buildNumber uint64
}

func NewChainConfigRoutine(
	cfg *config.ChainConfigConfig,
	chainID string,
	nodeSigner auth.Signer,
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
		logger:      goloom.NewLoomLogger(cfg.LogLevel, cfg.LogDestination),
		buildNumber: build,
	}, nil
}

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

	cc.Run()
}

func (cc *ChainConfigRoutine) Run() {
	for {
		dappClient := client.NewDAppChainRPCClient(cc.chainID, cc.cfg.DAppChainWriteURI, cc.cfg.DAppChainReadURI)
		chainConfigClient, err := NewChainConfigClient(dappClient, cc.address, cc.signer, cc.logger)
		if err != nil {
			cc.logger.Error("Failed to create ChainConfigClient", "err", err)
		} else {
			if err := chainConfigClient.VoteToEnablePendingFeatures(cc.buildNumber); err != nil {
				cc.logger.Error("Failed to auto-enable features", "err", err)
			}
		}
		time.Sleep(time.Duration(cc.cfg.EnableFeatureInterval) * time.Second)
	}
}
