// +build evm

package oracle

import (
	"log"
	"math/big"
	"runtime"
	"sort"
	"time"

	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/client/plasma_cash/eth"
	"github.com/pkg/errors"
)

const (
	DefaultMaxRetry   = 5
	DefaultRetryDelay = 1 * time.Second
)

type sortableRequests struct {
	requests []*pctypes.PlasmaCashRequest
}

func (s sortableRequests) Less(i, j int) bool {
	if s.requests[i].Meta.BlockNumber != s.requests[j].Meta.BlockNumber {
		return s.requests[i].Meta.BlockNumber < s.requests[j].Meta.BlockNumber
	}

	if s.requests[i].Meta.TxIndex != s.requests[j].Meta.TxIndex {
		return s.requests[i].Meta.TxIndex < s.requests[j].Meta.TxIndex
	}

	if s.requests[i].Meta.LogIndex != s.requests[j].Meta.LogIndex {
		return s.requests[i].Meta.LogIndex < s.requests[j].Meta.LogIndex
	}

	return i < j
}

func (s sortableRequests) Len() int {
	return len(s.requests)
}

func (s sortableRequests) Swap(i, j int) {
	tmpRequest := s.requests[i]
	s.requests[i] = s.requests[j]
	s.requests[j] = tmpRequest
}

func (s sortableRequests) PrepareRequestBatch() *pctypes.PlasmaCashRequestBatch {
	requestBatch := &pctypes.PlasmaCashRequestBatch{}

	sort.Sort(s)
	requestBatch.Requests = s.requests

	return requestBatch
}

type OracleConfig struct {
	// Each Plasma block number must be a multiple of this value
	PlasmaBlockInterval uint32
	DAppChainClientCfg  DAppChainPlasmaClientConfig
	EthClientCfg        eth.EthPlasmaClientConfig
}

type PlasmaBlockWorkerStatus struct {
	LastSeenDAppChainPlasmaBlockNum *big.Int
	LastSeenEthPlasmaBlockNum       *big.Int

	// Just to avoid hassle of looking into yaml file
	PlasmaBlockInterval uint32
}

// PlasmaBlockWorker sends non-deposit Plasma block from the DAppChain to Ethereum.
type PlasmaBlockWorker struct {
	ethPlasmaClient     eth.EthPlasmaClient
	dappPlasmaClient    DAppChainPlasmaClient
	plasmaBlockInterval uint32
	status              *PlasmaBlockWorkerStatus
}

func NewPlasmaBlockWorker(cfg *OracleConfig) *PlasmaBlockWorker {
	return &PlasmaBlockWorker{
		ethPlasmaClient:     &eth.EthPlasmaClientImpl{EthPlasmaClientConfig: cfg.EthClientCfg},
		dappPlasmaClient:    &DAppChainPlasmaClientImpl{DAppChainPlasmaClientConfig: cfg.DAppChainClientCfg},
		plasmaBlockInterval: cfg.PlasmaBlockInterval,
		status: &PlasmaBlockWorkerStatus{
			PlasmaBlockInterval: cfg.PlasmaBlockInterval,
		},
	}
}

func (w *PlasmaBlockWorker) Init() error {
	if err := w.ethPlasmaClient.Init(); err != nil {
		return err
	}
	return w.dappPlasmaClient.Init()
}

func (w *PlasmaBlockWorker) Status() *PlasmaBlockWorkerStatus {
	return w.status
}

func (w *PlasmaBlockWorker) Run() {
	go runWithRecovery(func() {
		loopWithInterval(w.sendPlasmaBlocksToEthereum, 5*time.Second)
	})
}

// DAppChain -> Plasma Blocks -> Ethereum
func (w *PlasmaBlockWorker) sendPlasmaBlocksToEthereum() error {
	pendingTxs, err := w.dappPlasmaClient.GetPendingTxs()
	if err != nil {
		return errors.Wrap(err, "failed to get pending transactions")
	}

	// Only call SubmitBlockToMainnet, if pending transactions are there.
	if len(pendingTxs.Transactions) > 0 {
		if err = w.dappPlasmaClient.FinalizeCurrentPlasmaBlock(); err != nil {
			return errors.Wrap(err, "failed to finalize current plasma block")
		}
	}

	if err = w.syncPlasmaBlocksWithEthereum(); err != nil {
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

	w.status.LastSeenEthPlasmaBlockNum = curEthPlasmaBlockNum

	log.Printf("solPlasma.CurrentBlock: %s", curEthPlasmaBlockNum.String())

	curLoomPlasmaBlockNum, err := w.dappPlasmaClient.CurrentPlasmaBlockNum()
	if err != nil {
		return err
	}

	w.status.LastSeenDAppChainPlasmaBlockNum = curLoomPlasmaBlockNum

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

type PlasmaCoinWorkerStatus struct {
	DepositEventsProcessed     int
	WithdrawEventsProcessed    int
	StartedExitEventsProcessed int
	CoinResetEventsProcessed   int

	LastSeenEthBlockNumber        uint64
	LastReportedRequestBatchTally *pctypes.PlasmaCashRequestBatchTally
}

// PlasmaCoinWorker sends Plasma deposits from Ethereum to the DAppChain.
type PlasmaCoinWorker struct {
	ethPlasmaClient  eth.EthPlasmaClient
	dappPlasmaClient DAppChainPlasmaClient
	status           *PlasmaCoinWorkerStatus
}

func NewPlasmaCoinWorker(cfg *OracleConfig) *PlasmaCoinWorker {
	return &PlasmaCoinWorker{
		ethPlasmaClient:  &eth.EthPlasmaClientImpl{EthPlasmaClientConfig: cfg.EthClientCfg},
		dappPlasmaClient: &DAppChainPlasmaClientImpl{DAppChainPlasmaClientConfig: cfg.DAppChainClientCfg},
		status:           &PlasmaCoinWorkerStatus{},
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

func (w *PlasmaCoinWorker) Status() *PlasmaCoinWorkerStatus {
	return w.status
}

func (w *PlasmaCoinWorker) sendCoinEventsToDAppChain() error {

	tally, err := w.dappPlasmaClient.GetRequestBatchTally()
	if err != nil {
		return errors.Wrapf(err, "failed to fetch current request batch tally from dappchain")
	}

	// If HasSeenAnyRequest is false means we havent seen any
	// block, so set startEthBlock to zero only, otherwise
	// set it to lastSeen + 1
	var startEthBlock uint64 = 0
	if tally.LastSeenBlockNumber != 0 {
		startEthBlock = tally.LastSeenBlockNumber + 1
	}

	// TODO: limit max block range per batch
	latestEthBlock, err := w.ethPlasmaClient.LatestEthBlockNum()
	if err != nil {
		return errors.Wrapf(err, "failed to fetch latest block number for eth contract")
	}

	w.status.LastSeenEthBlockNumber = latestEthBlock
	w.status.LastReportedRequestBatchTally = tally

	if latestEthBlock < startEthBlock {
		// Wait for Ethereum to produce a new block...
		return nil
	}

	// We need to retreive all events first, and then apply them in correct order
	// to make sure, we apply events in proper order to dappchain

	depositEvents, err := w.ethPlasmaClient.FetchDeposits(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma deposit events from Ethereum")
	}

	withdrewEvents, err := w.ethPlasmaClient.FetchWithdrews(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma withdrew events from Ethereum")
	}

	startedExitEvents, err := w.ethPlasmaClient.FetchStartedExit(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma started exit event from Ethereum")
	}

	coinResetEvents, err := w.ethPlasmaClient.FetchCoinReset(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma coin reset event from Ethereum")
	}

	requests := make([]*pctypes.PlasmaCashRequest, len(depositEvents)+len(withdrewEvents)+len(startedExitEvents)+len(coinResetEvents))

	i := 0
	for _, event := range depositEvents {
		requests[i] = &pctypes.PlasmaCashRequest{
			Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
				Slot:         event.Slot,
				DepositBlock: event.DepositBlock,
				Denomination: event.Denomination,
				From:         event.From,
				Contract:     event.Contract,
			}},
			Meta: event.Meta,
		}
		i++
	}
	for _, event := range withdrewEvents {
		requests[i] = &pctypes.PlasmaCashRequest{
			Data: &pctypes.PlasmaCashRequest_Withdraw{&pctypes.PlasmaCashWithdrawCoinRequest{
				Owner: event.Owner,
				Slot:  event.Slot,
			}},
			Meta: event.Meta,
		}
		i++
	}
	for _, event := range startedExitEvents {
		requests[i] = &pctypes.PlasmaCashRequest{
			Data: &pctypes.PlasmaCashRequest_StartedExit{&pctypes.PlasmaCashExitCoinRequest{
				Owner: event.Owner,
				Slot:  event.Slot,
			}},
			Meta: event.Meta,
		}
		i++
	}
	for _, event := range coinResetEvents {
		requests[i] = &pctypes.PlasmaCashRequest{
			Data: &pctypes.PlasmaCashRequest_CoinReset{&pctypes.PlasmaCashCoinResetRequest{
				Owner: event.Owner,
				Slot:  event.Slot,
			}},
			Meta: event.Meta,
		}
		i++
	}

	// No requests to process
	if len(requests) == 0 {
		return nil
	}

	requestBatch := sortableRequests{requests: requests}.PrepareRequestBatch()
	err = w.dappPlasmaClient.ProcessRequestBatch(requestBatch)
	if err != nil {
		return errors.Wrapf(err, "unable to send request batch to dappchain")
	}

	w.status.DepositEventsProcessed += len(depositEvents)
	w.status.WithdrawEventsProcessed += len(withdrewEvents)
	w.status.StartedExitEventsProcessed += len(startedExitEvents)
	w.status.CoinResetEventsProcessed += len(coinResetEvents)

	return nil

}

type OracleStatus struct {
	CoinWorkerStatus  *PlasmaCoinWorkerStatus
	BlockWorkerStatus *PlasmaBlockWorkerStatus
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

func (orc *Oracle) Status() *OracleStatus {
	return &OracleStatus{
		CoinWorkerStatus:  orc.coinWorker.Status(),
		BlockWorkerStatus: orc.blockWorker.Status(),
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
		counter := 0
		loopWithInterval(func() error {
			counter += 1
			if counter == 6 { // Submit blocks 6 times less often than fetching events (12 sec)
				err := orc.blockWorker.sendPlasmaBlocksToEthereum()
				if err != nil {
					log.Printf("[PCOracle] error while sending plasma blocks to ethereum: %v\n", err)
				}
				counter = 0
			}

			err := orc.coinWorker.sendCoinEventsToDAppChain()
			if err != nil {
				log.Printf("[PCOracle] error while sending coin events to dappchain: %v\n", err)
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
