// +build evm

package oracle

import (
	"context"
	"crypto/ecdsa"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/loomnetwork/go-loom/auth"

	"github.com/loomnetwork/go-loom/client/dposv2"
	"github.com/loomnetwork/go-loom/client/timelock"

	"github.com/loomnetwork/go-loom/client"

	loom "github.com/loomnetwork/go-loom"

	"github.com/loomnetwork/go-loom/types"

	"github.com/ethereum/go-ethereum/ethclient"

	d2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"

	"encoding/base64"
)

type DAppChainDPOSv2ClientConfig struct {
	ChainID  string
	WriteURI string
	ReadURI  string
	// Used to sign txs sent to Loom DAppChain
	Signer auth.Signer
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
}

type TimeLockWorkerStatus struct {
	Enabled                       bool
	TimeLockFactoryHexAddress     string
	NumberOfTimeLockEventsFetched uint64
}

type Config struct {
	Enabled              bool
	StatusServiceAddress string
	DAppChainClientCfg   DAppChainDPOSv2ClientConfig
	EthClientCfg         EthClientConfig
	MainnetPollInterval  time.Duration
	TimeLockWorkerCfg    TimeLockWorkerConfig
}

type Status struct {
	TimeLock                TimeLockWorkerStatus
	OracleAddress           string
	NumberOfEventsSubmitted uint64
}

type timeLockWorker struct {
	cfg                   *TimeLockWorkerConfig
	chainID               string
	timelockFactoryClient *timelock.MainnetTimelockFactoryClient

	statusRWMutex sync.RWMutex
	status        TimeLockWorkerStatus
}

func newTimeLockWorker(cfg *TimeLockWorkerConfig) *timeLockWorker {
	return &timeLockWorker{
		cfg: cfg,
	}
}

func (t *timeLockWorker) Status() TimeLockWorkerStatus {
	t.statusRWMutex.RLock()
	defer t.statusRWMutex.RUnlock()

	return TimeLockWorkerStatus{
		Enabled:                       t.status.Enabled,
		TimeLockFactoryHexAddress:     t.status.TimeLockFactoryHexAddress,
		NumberOfTimeLockEventsFetched: t.status.NumberOfTimeLockEventsFetched,
	}
}

func (t *timeLockWorker) Init(chainID string, mainnetClient *ethclient.Client) error {
	if !t.cfg.Enabled {
		return nil
	}

	timelockFactoryClient, err := timelock.ConnectToMainnetTimelockFactory(mainnetClient, t.cfg.TimeLockFactoryHexAddress)
	if err != nil {
		return err
	}
	t.timelockFactoryClient = timelockFactoryClient
	t.chainID = chainID

	t.status.TimeLockFactoryHexAddress = t.cfg.TimeLockFactoryHexAddress
	t.status.Enabled = t.cfg.Enabled

	return nil
}

func (t *timeLockWorker) FetchRequestBatch(identity *client.Identity, tally *d2types.RequestBatchTallyV2, latestBlock uint64) ([]*d2types.BatchRequestV2, error) {
	if !t.cfg.Enabled {
		return nil, nil
	}

	tokenTimeLockCreationEvents, err := t.timelockFactoryClient.FetchTokenTimeLockCreationEvent(identity, tally.LastSeenBlockNumber+1, latestBlock)
	if err != nil {
		return nil, err
	}

	requestBatch := make([]*d2types.BatchRequestV2, len(tokenTimeLockCreationEvents))

	for i, event := range tokenTimeLockCreationEvents {
		candidatePubKey, err := base64.StdEncoding.DecodeString(event.ValidatorPublicKey)
		if err != nil {
			return nil, err
		}

		candidateLocalAddress := loom.LocalAddressFromPublicKey(candidatePubKey)

		requestBatch[i] = &d2types.BatchRequestV2{
			Payload: &d2types.BatchRequestV2_WhitelistCandidate{&d2types.WhitelistCandidateRequestV2{
				CandidateAddress: &types.Address{
					Local:   candidateLocalAddress,
					ChainId: t.chainID,
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

	t.statusRWMutex.Lock()
	t.status.NumberOfTimeLockEventsFetched += uint64(len(requestBatch))
	t.statusRWMutex.Unlock()

	return requestBatch, nil
}

type Oracle struct {
	cfg *Config

	dappchainRPCClient *client.DAppChainRPCClient
	mainnetClient      *ethclient.Client
	dposContract       *dposv2.DAppChainDPOSContract

	identity *client.Identity

	// Workers
	timelockWorker *timeLockWorker

	statusRWMutex sync.RWMutex
	status        Status
}

func NewOracle(cfg *Config) *Oracle {
	return &Oracle{
		cfg: cfg,
	}
}

func (o *Oracle) Init() error {
	o.identity = &client.Identity{
		MainnetPrivKey: o.cfg.EthClientCfg.PrivateKey,
		LoomSigner:     o.cfg.DAppChainClientCfg.Signer,
		LoomAddr: loom.Address{
			ChainID: o.cfg.DAppChainClientCfg.ChainID,
			Local:   loom.LocalAddressFromPublicKey(o.cfg.DAppChainClientCfg.Signer.PublicKey()),
		},
	}

	dppchainRPCClient := client.NewDAppChainRPCClient(o.cfg.DAppChainClientCfg.ChainID, o.cfg.DAppChainClientCfg.WriteURI, o.cfg.DAppChainClientCfg.ReadURI)
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

	o.timelockWorker = newTimeLockWorker(&o.cfg.TimeLockWorkerCfg)
	if err = o.timelockWorker.Init(o.cfg.DAppChainClientCfg.ChainID, mainnetClient); err != nil {
		return err
	}

	o.status.OracleAddress = o.identity.LoomAddr.String()

	return nil
}

func (o *Oracle) Status() Status {
	o.statusRWMutex.RLock()
	defer o.statusRWMutex.RUnlock()

	return Status{
		TimeLock:                o.timelockWorker.Status(),
		OracleAddress:           o.status.OracleAddress,
		NumberOfEventsSubmitted: o.status.NumberOfEventsSubmitted,
	}
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

	var projectedRequestCount = 0

	// Fetch events from worker
	tokenTimeLockCreationEvents, err := o.timelockWorker.FetchRequestBatch(o.identity, tally, latestBlock)
	if err != nil {
		return err
	}
	projectedRequestCount += len(tokenTimeLockCreationEvents)

	if projectedRequestCount == 0 {
		return nil
	}

	requestBatch := make([]*d2types.BatchRequestV2, 0, projectedRequestCount)
	requestBatch = append(requestBatch, tokenTimeLockCreationEvents...)

	if err := o.dposContract.ProcessRequestBatch(o.identity, &d2types.RequestBatchV2{
		Batch: requestBatch,
	}); err != nil {
		return err
	}

	o.statusRWMutex.Lock()
	o.status.NumberOfEventsSubmitted += uint64(len(requestBatch))
	o.statusRWMutex.Unlock()

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
