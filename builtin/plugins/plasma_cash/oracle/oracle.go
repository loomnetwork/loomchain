// +build evm

package oracle

import (
	"log"
	"math/big"
	"runtime"
	"time"

	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/client/plasma_cash/eth"
	"github.com/pkg/errors"
)

type OracleConfig struct {
	// Each Plasma block number must be a multiple of this value
	PlasmaBlockInterval uint32
	DAppChainClientCfg  DAppChainPlasmaClientConfig
	EthClientCfg        eth.EthPlasmaClientConfig
}

// PlasmaBlockWorker sends non-deposit Plasma block from the DAppChain to Ethereum.
type PlasmaBlockWorker struct {
	ethPlasmaClient     eth.EthPlasmaClient
	dappPlasmaClient    DAppChainPlasmaClient
	plasmaBlockInterval uint32
}

func NewPlasmaBlockWorker(cfg *OracleConfig) *PlasmaBlockWorker {
	return &PlasmaBlockWorker{
		ethPlasmaClient:     &eth.EthPlasmaClientImpl{EthPlasmaClientConfig: cfg.EthClientCfg},
		dappPlasmaClient:    &DAppChainPlasmaClientImpl{DAppChainPlasmaClientConfig: cfg.DAppChainClientCfg},
		plasmaBlockInterval: cfg.PlasmaBlockInterval,
	}
}

func (w *PlasmaBlockWorker) Init() error {
	if err := w.ethPlasmaClient.Init(); err != nil {
		return err
	}
	return w.dappPlasmaClient.Init()
}

func (w *PlasmaBlockWorker) Run() {
	go runWithRecovery(func() {
		loopWithInterval(w.sendPlasmaBlocksToEthereum, 5*time.Second)
	})
}

// DAppChain -> Plasma Blocks -> Ethereum
func (w *PlasmaBlockWorker) sendPlasmaBlocksToEthereum() error {
	w.dappPlasmaClient.FinalizeCurrentPlasmaBlock()
	if err := w.syncPlasmaBlocksWithEthereum(); err != nil {
		return errors.Wrap(err, "failed to sync plasma blocks with mainnet")
	}
	return nil

}

// Send any finalized but unsubmitted plasma blocks from the DAppChain to Ethereum.
func (w *PlasmaBlockWorker) syncPlasmaBlocksWithEthereum() error {
	curEthPlasmaBlockNum, err := w.ethPlasmaClient.CurrentPlasmaBlockNum()
	if err != nil {
		return err
	}
	log.Printf("solPlasma.CurrentBlock: %s", curEthPlasmaBlockNum.String())

	curLoomPlasmaBlockNum, err := w.dappPlasmaClient.CurrentPlasmaBlockNum()
	if err != nil {
		return err
	}

	if curLoomPlasmaBlockNum.Cmp(curEthPlasmaBlockNum) == 0 {
		// DAppChain and Ethereum both have all the finalized Plasma blocks
		return nil
	}

	plasmaBlockInterval := big.NewInt(int64(w.plasmaBlockInterval))
	unsubmittedPlasmaBlockNum := nextPlasmaBlockNum(curEthPlasmaBlockNum, plasmaBlockInterval)

	if unsubmittedPlasmaBlockNum.Cmp(curLoomPlasmaBlockNum) > 0 {
		// All the finalized plasma blocks in the DAppChain have been submitted to Ethereum
		return nil
	}

	block, err := w.dappPlasmaClient.PlasmaBlockAt(unsubmittedPlasmaBlockNum)
	if err != nil {
		return err
	}

	if err := w.submitPlasmaBlockToEthereum(unsubmittedPlasmaBlockNum, block.MerkleHash); err != nil {
		return err
	}

	return nil
}

// Submits a Plasma block (or rather its merkle root) to the Plasma Solidity contract on Ethereum.
// This function will block until the tx is confirmed, or times out.
func (w *PlasmaBlockWorker) submitPlasmaBlockToEthereum(plasmaBlockNum *big.Int, merkleRoot []byte) error {
	curEthPlasmaBlockNum, err := w.ethPlasmaClient.CurrentPlasmaBlockNum()
	if err != nil {
		return err
	}

	// Try to avoid submitting the same plasma blocks multiple times
	if plasmaBlockNum.Cmp(curEthPlasmaBlockNum) <= 0 {
		return nil
	}

	if len(merkleRoot) != 32 {
		return errors.New("invalid merkle root size")
	}

	var root [32]byte
	copy(root[:], merkleRoot)
	log.Printf("********* #### Submitting plasmaBlockNum: %s with root: %v", plasmaBlockNum.String(), root)
	return w.ethPlasmaClient.SubmitPlasmaBlock(plasmaBlockNum, root)
}

// PlasmaCoinWorker sends Plasma deposits from Ethereum to the DAppChain.
type PlasmaCoinWorker struct {
	ethPlasmaClient  eth.EthPlasmaClient
	dappPlasmaClient DAppChainPlasmaClient
	startEthBlock    uint64 // Eth block from which the oracle should start looking for deposits
}

func NewPlasmaCoinWorker(cfg *OracleConfig) *PlasmaCoinWorker {
	return &PlasmaCoinWorker{
		ethPlasmaClient:  &eth.EthPlasmaClientImpl{EthPlasmaClientConfig: cfg.EthClientCfg},
		dappPlasmaClient: &DAppChainPlasmaClientImpl{DAppChainPlasmaClientConfig: cfg.DAppChainClientCfg},
	}
}

func (w *PlasmaCoinWorker) Init() error {
	if err := w.ethPlasmaClient.Init(); err != nil {
		return err
	}
	return w.dappPlasmaClient.Init()
}

func (w *PlasmaCoinWorker) Run() {
	go runWithRecovery(func() {
		loopWithInterval(w.sendCoinEventsToDAppChain, 4*time.Second)
	})
}

func (w *PlasmaCoinWorker) sendCoinEventsToDAppChain() error {
	// TODO: get start block from Plasma Go contract, like the Transfer Gateway Oracle
	startEthBlock := w.startEthBlock

	// TODO: limit max block range per batch
	latestEthBlock, err := w.ethPlasmaClient.LatestEthBlockNum()
	if err != nil {
		return errors.Wrapf(err, "failed to fetch latest block number for eth contract")
	}

	if latestEthBlock < startEthBlock {
		// Wait for Ethereum to produce a new block...
		return nil
	}

	// We need to retreive all events first, and then apply them in correct order
	// to make sure, we apply events in proper order to dappchain

	unSubmittedDepositeEvents, err := w.ethPlasmaClient.FetchDeposits(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma deposit events from Ethereum")
	}

	unSubmittedWithdrewEvents, err := w.ethPlasmaClient.FetchWithdrews(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma withdrew events from Ethereum")
	}

	unSubmittedStartedExitEvents, err := w.ethPlasmaClient.FetchStartedExit(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma started exit event from Ethereum")
	}

	unsubmittedCoinResetEvents, err := w.ethPlasmaClient.FetchCoinReset(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma coin reset event from Ethereum")
	}

	// Events will always be submitted in correct order. If submitting an event fails,
	// it will be resumed from there in next iteration.
	for len(unSubmittedDepositeEvents) != 0 || len(unSubmittedStartedExitEvents) != 0 || len(unSubmittedWithdrewEvents) != 0 {
		unSubmittedDepositeEvents, err = w.sendPlasmaDepositEventsToDAppChain(unSubmittedDepositeEvents)
		if err != nil {
			log.Printf("failed to send plasma deposit events to dappchain. Error: %v", err)
			continue
		}

		unSubmittedStartedExitEvents, err = w.sendPlasmaStartedExitEventsToDAppChain(unSubmittedStartedExitEvents)
		if err != nil {
			log.Printf("failed to send plasma start exit events to dappchain. Error: %v", err)
			continue
		}

		unSubmittedWithdrewEvents, err = w.sendPlasmaWithdrewEventsToDAppChain(unSubmittedWithdrewEvents)
		if err != nil {
			log.Printf("failed to send plasma withdraw events to dappchain. Error: %v", err)
			continue
		}

		unsubmittedCoinResetEvents, err = w.sendPlasmaCoinResetEventsToDAppChain(coinResetEvents)
		if err != nil {
			log.Printf("failed to send plasma coin reset events to dappchain. Error: %v", err)
			continue
		}
	}

	w.startEthBlock = latestEthBlock + 1

	return nil

}

func (w *PlasmaCoinWorker) sendPlasmaCoinResetEventsToDAppChain(coinResetEvents []*pctypes.PlasmaCashCoinResetEvent) ([]*pctypes.PlasmaCashCoinResetEvent, error) {
	for i, coinResetEvent := range coinResetEvents {
		if err := w.dappPlasmaClient.Reset(&pctypes.PlasmaCashCoinResetRequest{
			Owner: coinResetEvent.Owner,
			Slot:  coinResetEvent.Slot,
		}); err != nil {
			return coinResetEvents[i:], err
		}
	}

	return nil, nil
}

func (w *PlasmaCoinWorker) sendPlasmaStartedExitEventsToDAppChain(startedExitEvents []*pctypes.PlasmaCashStartedExitEvent) ([]*pctypes.PlasmaCashStartedExitEvent, error) {
	for i, startedExitEvent := range startedExitEvents {
		if err := w.dappPlasmaClient.Exit(&pctypes.PlasmaCashExitCoinRequest{
			Owner: startedExitEvent.Owner,
			Slot:  startedExitEvent.Slot,
		}); err != nil {
			return startedExitEvents[i:], err
		}
	}

	return nil, nil
}

func (w *PlasmaCoinWorker) sendPlasmaWithdrewEventsToDAppChain(withdrewEvents []*pctypes.PlasmaCashWithdrewEvent) ([]*pctypes.PlasmaCashWithdrewEvent, error) {
	for i, withdrewEvent := range withdrewEvents {
		if err := w.dappPlasmaClient.Withdraw(&pctypes.PlasmaCashWithdrawCoinRequest{
			Owner: withdrewEvent.Owner,
			Slot:  withdrewEvent.Slot,
		}); err != nil {
			return withdrewEvents[i:], err
		}
	}

	return nil, nil
}

// Ethereum -> Plasma Deposits -> DAppChain
func (w *PlasmaCoinWorker) sendPlasmaDepositEventsToDAppChain(depositEvents []*pctypes.PlasmaDepositEvent) ([]*pctypes.PlasmaDepositEvent, error) {

	for i, depositEvent := range depositEvents {
		if err := w.dappPlasmaClient.Deposit(&pctypes.DepositRequest{
			Slot:         depositEvent.Slot,
			DepositBlock: depositEvent.DepositBlock,
			Denomination: depositEvent.Denomination,
			From:         depositEvent.From,
			Contract:     depositEvent.Contract,
		}); err != nil {
			return depositEvents[i:], err
		}
	}

	return nil, nil
}

type Oracle struct {
	cfg         *OracleConfig
	coinWorker  *PlasmaCoinWorker
	blockWorker *PlasmaBlockWorker
}

func NewOracle(cfg *OracleConfig) *Oracle {
	return &Oracle{
		cfg:         cfg,
		coinWorker:  NewPlasmaCoinWorker(cfg),
		blockWorker: NewPlasmaBlockWorker(cfg),
	}
}

func (orc *Oracle) Init() error {
	if err := orc.coinWorker.Init(); err != nil {
		return err
	}
	return orc.blockWorker.Init()
}

// TODO: Graceful shutdown
func (orc *Oracle) Run() {
	go runWithRecovery(func() {
		loopWithInterval(func() error {
			err := orc.blockWorker.sendPlasmaBlocksToEthereum()
			if err != nil {
				log.Printf("error while sending plasma blocks to ethereum: %v\n", err)
			}

			err = orc.coinWorker.sendCoinEventsToDAppChain()
			if err != nil {
				log.Printf("error while sending coin events to dappchain: %v\n", err)
			}

			return err
		}, 2*time.Second)
	})
}

// runWithRecovery should run in a goroutine, it will ensure the given function keeps on running in
// a goroutine as long as it doesn't panic due to a runtime error.
func runWithRecovery(run func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in a Plasma Oracle worker: %v\n", r)
			// Unless it's a runtime error restart the goroutine
			if _, ok := r.(runtime.Error); !ok {
				time.Sleep(30 * time.Second)
				log.Printf("Restarting Plasma Oracle worker...")
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

// TODO: This function should be moved to loomchain/builtin/plasma_cash when the Oracle is
//       integrated into loomchain.
// Computes the block number of the next non-deposit Plasma block.
// The current Plasma block number can be for a deposit or non-deposit Plasma block.
// Plasma block numbers of non-deposit blocks are expected to be multiples of the specified interval.
func nextPlasmaBlockNum(current *big.Int, interval *big.Int) *big.Int {
	r := current
	r.Div(r, interval)
	r.Add(r, big.NewInt(1))
	return r.Mul(r, interval)
}
