// +build evm

package oracle

import (
	"context"
	"crypto/ecdsa"
	"log"
	"runtime"
	"time"

	"github.com/loomnetwork/go-loom/auth"

	"github.com/loomnetwork/go-loom/client/dposv2"
	"github.com/loomnetwork/go-loom/client/timelock"

	"github.com/loomnetwork/go-loom/client"

	loom "github.com/loomnetwork/go-loom"

	"github.com/loomnetwork/go-loom/types"

	"github.com/ethereum/go-ethereum/ethclient"

	d2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"
)

type DAppChainDPOSv2ClientConfig struct {
	ChainID  string
	WriteURI string
	ReadURI  string
	// Used to sign txs sent to Loom DAppChain
	Signer auth.Signer
	// name of dposv2 contract on DAppChain
	ContractName string
}

type EthClientConfig struct {
	// URI of an Ethereum node
	EthereumURI string
	// Private key that should be used to sign txs sent to Ethereum
	PrivateKey *ecdsa.PrivateKey
}

type TimeLockWorkerConfig struct {
	TimeLockFactoryHexAddress string
	Enabled                   bool
	startBlock                uint64
	endBlock                  uint64
}

type Config struct {
	Enabled              bool
	StatusServiceAddress string
	DAppChainClientCfg   DAppChainDPOSv2ClientConfig
	EthClientCfg         EthClientConfig
	MainnetPollInterval  time.Duration
	TimeLockWorkerCfg    TimeLockWorkerConfig
}

type Oracle struct {
	cfg *Config

	dappchainRPCClient *client.DAppChainRPCClient
	mainnetClient      *ethclient.Client

	timelockFactoryClient *timelock.MainnetTimelockFactoryClient
	dposContract          *dposv2.DAppChainDPOSContract

	identity *client.Identity
}

func NewOracle(cfg *Config) *Oracle {
	return &Oracle{
		cfg: cfg,
	}
}

func (o *Oracle) Init(chainID string) error {
	o.identity = &client.Identity{
		MainnetPrivKey: o.cfg.EthClientCfg.PrivateKey,
		LoomSigner:     o.cfg.DAppChainClientCfg.Signer,
		LoomAddr: loom.Address{
			ChainID: chainID,
			Local:   loom.LocalAddressFromPublicKeyV2(o.cfg.DAppChainClientCfg.Signer.PublicKey()),
		},
	}

	dppchainRPCClient := client.NewDAppChainRPCClient(chainID, o.cfg.DAppChainClientCfg.WriteURI, o.cfg.DAppChainClientCfg.ReadURI)
	o.dappchainRPCClient = dppchainRPCClient

	mainnetClient, err := ethclient.Dial(o.cfg.EthClientCfg.EthereumURI)
	if err != nil {
		return err
	}
	o.mainnetClient = mainnetClient

	dposContract, err := dposv2.ConnectToDAppChainDPOSContract(dppchainRPCClient)
	if err != nil {
		return err
	}
	o.dposContract = dposContract

	if o.cfg.TimeLockWorkerCfg.Enabled {
		timelockFactoryClient, err := timelock.ConnectToMainnetTimelockFactory(mainnetClient, o.cfg.TimeLockWorkerCfg.TimeLockFactoryHexAddress)
		if err != nil {
			return err
		}
		o.timelockFactoryClient = timelockFactoryClient
	}

	return nil
}

func (o *Oracle) process() error {
	// Get latest block number
	blockHeader, err := o.mainnetClient.HeaderByNumber(context.TODO(), nil)
	if err != nil {
		return err
	}
	latestBlock := blockHeader.Number.Uint64()

	tally, err := o.dposContract.GetRequestBatchTally(o.identity)
	if err != nil {
		return err
	}

	var tokenTimeLockCreationEvents []*timelock.LoomTimelockFactoryLoomTimeLockCreated
	var projectedRequestCount = 0

	if o.cfg.TimeLockWorkerCfg.Enabled {
		// Fetch token time lock creation event between startBlock and latestBlock
		tokenTimeLockCreationEvents, err = o.timelockFactoryClient.FetchTokenTimeLockCreationEvent(o.identity, tally.LastSeenBlockNumber, latestBlock)
		if err != nil {
			return err
		}
		projectedRequestCount += len(tokenTimeLockCreationEvents)
	}

	if projectedRequestCount == 0 {
		return nil
	}

	requestBatch := make([]*d2types.BatchRequestV2, projectedRequestCount)

	for i, event := range tokenTimeLockCreationEvents {
		candidateLocalAddress, err := loom.LocalAddressFromHexString(event.ValidatorEthAddress.Hex())
		if err != nil {
			return err
		}

		requestBatch[i] = &d2types.BatchRequestV2{
			Payload: &d2types.BatchRequestV2_WhitelistCandidate{&d2types.WhitelistCandidateRequestV2{
				CandidateAddress: &types.Address{
					Local:   candidateLocalAddress,
					ChainId: "eth",
				},
				Amount:   &types.BigUInt{Value: *loom.NewBigUInt(event.Amount)},
				LockTime: event.ReleaseTime.Uint64(),
			}},
			Meta: &d2types.BatchRequestMetaV2{
				BlockNumber: event.Raw.BlockNumber,
				TxIndex:     uint64(event.Raw.TxIndex),
				LogIndex:    uint64(event.Raw.Index),
			},
		}
	}

	if err := o.dposContract.ProcessRequestBatch(o.identity, &d2types.RequestBatchV2{
		Batch: requestBatch,
	}); err != nil {
		return err
	}

	return nil

}

func (o *Oracle) Run() {
	go runWithRecovery(func() {
		loopWithInterval(o.process, o.cfg.MainnetPollInterval)
	})
}

// runWithRecovery should run in a goroutine, it will ensure the given function keeps on running in
// a goroutine as long as it doesn't panic due to a runtime error.
func runWithRecovery(run func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in a DPOSV2 Oracle worker: %v\n", r)
			// Unless it's a runtime error restart the goroutine
			if _, ok := r.(runtime.Error); !ok {
				time.Sleep(30 * time.Second)
				log.Printf("Restarting DPOSV2 Oracle worker...")
				go runWithRecovery(run)
			}
		}
	}()
	run()
}

// loopWithInterval will execute the step function in an endless loop while ensuring that each
// loop iteration takes up the minimum specified duration.
func loopWithInterval(step func() error, minStepDuration time.Duration) {
	for {
		start := time.Now()
		if err := step(); err != nil {
			log.Println(err)
		}
		diff := time.Now().Sub(start)
		if diff < minStepDuration {
			time.Sleep(minStepDuration - diff)
		}
	}
}
