// +build evm

package oracle

import (
	"fmt"
	"log"
	"math/big"
	"runtime"
	"time"

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
	w.syncPlasmaBlocksWithEthereum()
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

	log.Printf("@@@@@@@@@@ CurrentPlasmaBlockNumber: %s, curEthPlasmaBlockNum: %s", curLoomPlasmaBlockNum.String(), curEthPlasmaBlockNum.String())

	if curLoomPlasmaBlockNum.Cmp(curEthPlasmaBlockNum) == 0 {
		// DAppChain and Ethereum both have all the finalized Plasma blocks
		return nil
	}

	plasmaBlockInterval := big.NewInt(int64(w.plasmaBlockInterval))
	unsubmittedPlasmaBlockNum := nextPlasmaBlockNum(curEthPlasmaBlockNum, plasmaBlockInterval)

	log.Printf("unsubmittedPlasmaBlockNum: %s, curLoomPlasmaBlockNum: %s, qazwsx", unsubmittedPlasmaBlockNum.String(), curLoomPlasmaBlockNum.String())

	if unsubmittedPlasmaBlockNum.Cmp(curLoomPlasmaBlockNum) > 0 {
		log.Printf("unsubmittedPlasmaBlockNum: %s, curLoomPlasmaBlockNum: %s, edcrfv", unsubmittedPlasmaBlockNum.String(), curLoomPlasmaBlockNum.String())
		// All the finalized plasma blocks in the DAppChain have been submitted to Ethereum
		fmt.Printf("******* All Blocks are submitted to ethereum ********\n", unsubmittedPlasmaBlockNum)
		return nil
	}

	block, err := w.dappPlasmaClient.PlasmaBlockAt(unsubmittedPlasmaBlockNum)
	if err != nil {
		return err
	}

	fmt.Printf("******* Unsubmitted block number: %d ********\n", unsubmittedPlasmaBlockNum)

	if err := w.submitPlasmaBlockToEthereum(unsubmittedPlasmaBlockNum, block.MerkleHash); err != nil {
		return err
	}

	fmt.Println("!!!!!!!!! PlasmaBlockInterval: %d", plasmaBlockInterval)

	return nil
}

// Submits a Plasma block (or rather its merkle root) to the Plasma Solidity contract on Ethereum.
// This function will block until the tx is confirmed, or times out.
func (w *PlasmaBlockWorker) submitPlasmaBlockToEthereum(plasmaBlockNum *big.Int, merkleRoot []byte) error {
	curEthPlasmaBlockNum, err := w.ethPlasmaClient.CurrentPlasmaBlockNum()
	if err != nil {
		return err
	}

	fmt.Printf("********** Current plasma block number: %d **********\n", curEthPlasmaBlockNum)

	// Try to avoid submitting the same plasma blocks multiple times
	if plasmaBlockNum.Cmp(curEthPlasmaBlockNum) <= 0 {
		fmt.Printf("********** Current plasma block number: %s is same as plasmaBlockNum: %s **********\n", curEthPlasmaBlockNum.String(), plasmaBlockNum.String())
		return nil
	}

	fmt.Printf("********** ##### Submitting plasmaBlockNum: %s to ethereum\n", plasmaBlockNum.String())

	if len(merkleRoot) != 32 {
		return errors.New("invalid merkle root size")
	}

	var root [32]byte
	copy(root[:], merkleRoot)
	fmt.Printf("********* #### Submitting plasmaBlockNum: %s with root: %v", plasmaBlockNum.String(), root)
	return w.ethPlasmaClient.SubmitPlasmaBlock(plasmaBlockNum, root)
}

// PlasmaDepositWorker sends Plasma deposits from Ethereum to the DAppChain.
type PlasmaDepositWorker struct {
	ethPlasmaClient  eth.EthPlasmaClient
	dappPlasmaClient DAppChainPlasmaClient
	startEthBlock    uint64 // Eth block from which the oracle should start looking for deposits
}

func NewPlasmaDepositWorker(cfg *OracleConfig) *PlasmaDepositWorker {
	return &PlasmaDepositWorker{
		ethPlasmaClient:  &eth.EthPlasmaClientImpl{EthPlasmaClientConfig: cfg.EthClientCfg},
		dappPlasmaClient: &DAppChainPlasmaClientImpl{DAppChainPlasmaClientConfig: cfg.DAppChainClientCfg},
	}
}

func (w *PlasmaDepositWorker) Init() error {
	if err := w.ethPlasmaClient.Init(); err != nil {
		return err
	}
	return w.dappPlasmaClient.Init()
}

func (w *PlasmaDepositWorker) Run() {
	go runWithRecovery(func() {
		loopWithInterval(w.sendPlasmaDepositsToDAppChain, 1*time.Second)
	})
}

// Ethereum -> Plasma Deposits -> DAppChain
func (w *PlasmaDepositWorker) sendPlasmaDepositsToDAppChain() error {
	// TODO: get start block from Plasma Go contract, like the Transfer Gateway Oracle
	startEthBlock := w.startEthBlock

	// TODO: limit max block range per batch
	latestEthBlock, err := w.ethPlasmaClient.LatestEthBlockNum()
	if err != nil {
		return errors.Wrap(err, "failed to obtain latest Ethereum block number")
	}

	if latestEthBlock < startEthBlock {
		// Wait for Ethereum to produce a new block...
		return nil
	}

	deposits, err := w.ethPlasmaClient.FetchDeposits(startEthBlock, latestEthBlock)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Plasma deposits from Ethereum")
	}

	for _, deposit := range deposits {
		if err := w.dappPlasmaClient.Deposit(deposit); err != nil {
			return err
		}
	}

	w.startEthBlock = latestEthBlock + 1
	return nil
}

type Oracle struct {
	cfg           *OracleConfig
	depositWorker *PlasmaDepositWorker
	blockWorker   *PlasmaBlockWorker
}

func NewOracle(cfg *OracleConfig) *Oracle {
	return &Oracle{
		cfg:           cfg,
		depositWorker: NewPlasmaDepositWorker(cfg),
		blockWorker:   NewPlasmaBlockWorker(cfg),
	}
}

func (orc *Oracle) Init() error {
	if err := orc.depositWorker.Init(); err != nil {
		return err
	}
	return orc.blockWorker.Init()
}

// TODO: Graceful shutdown
func (orc *Oracle) Run() {
	if orc.cfg.EthClientCfg.PrivateKey != nil {
		orc.blockWorker.Run()
	}
	orc.depositWorker.Run()
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
	if current.Cmp(new(big.Int)) == 0 {
		//fmt.Println("chkpnt1 !@!", current.String(), interval.String(), new(big.Int).Set(interval).String())
		return new(big.Int).Set(interval)
	}
	if current.Cmp(interval) == 0 {
		//fmt.Println("chkpnt2 !@!", current.String(), interval.String(), new(big.Int).Add(current, interval).String())
		return new(big.Int).Add(current, interval)
	}

	r := new(big.Int).Add(current, big.NewInt(0))
	r.Div(r, interval)
	r.Add(r, big.NewInt(1))
	r.Mul(r, interval)
	fmt.Println("!@@! 1", current.String())
	fmt.Println("!@@! 3", r.String())
	return r
	/**
	r := new(big.Int).Add(current, new(big.Int).Sub(interval, big.NewInt(1)))
	//fmt.Println("chkpnt3 !@!", current.String(), interval.String(), new(big.Int).Sub(interval, big.NewInt(1)).String())
	//fmt.Println("chkpnt4", r.String())
	fmt.Println("!@!", r.String())
	r.Div(r, interval)
	r.Add(r, big.NewInt(1))

	//fmt.Println("chkpnt5 !@!", r.String())
	//fmt.Println("chkpnt6 !@!", r.Mul(r, interval).String()) // (current + (interval - 1)) / interval
	ans := r.Mul(r, interval)
	return ans // ((current + (interval - 1)) / interval) * interval
	**/
}
