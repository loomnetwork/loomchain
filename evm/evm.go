// +build evm

package evm

import (
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"fmt"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
)

var (
	gasLimit = uint64(math.MaxUint64)
	value    = new(big.Int)
)

//Metrics
var (
	txLatency metrics.Histogram
	txGas     metrics.Histogram
	txCount   metrics.Counter
)

func init() {
	fieldKeys := []string{"method", "error"}
	txCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "evm_transaction_count",
		Help:      "Number of evm transactions received.",
	}, fieldKeys)
	txLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "evmtx_latency_microseconds",
		Help:      "Total duration of go-ethereum EVM tx in microseconds.",
	}, fieldKeys)
	txGas = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "application",
		Name:      "evm_tx_gas_cost",
		Help:      "Gas cost of EVM transaction.",
	}, fieldKeys)
}

// TODO: this shouldn't be exported, rename to wrappedEVM
type Evm struct {
	state       state.StateDB
	context     vm.Context
	chainConfig params.ChainConfig
	vmConfig    vm.Config
}

func NewMockEvm() *Evm {
	p := new(Evm)
	db := ethdb.NewMemDatabase()
	_state, _ := state.New(common.Hash{}, state.NewDatabase(db))
	p.state = *_state
	p.chainConfig = defaultChainConfig()
	p.vmConfig = defaultVmConfig()
	p.context = defaultContext()
	return p
}

func NewEvm(_state state.StateDB, lstate loomchain.StoreState) *Evm {
	p := new(Evm)
	p.state = _state
	p.chainConfig = defaultChainConfig()
	p.vmConfig = defaultVmConfig()
	p.context = vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash: func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		},
		Coinbase:    common.BytesToAddress([]byte("myCoinBase")),
		BlockNumber: big.NewInt(lstate.Block().Height),
		Time:        big.NewInt(lstate.Block().Time),
		Difficulty:  new(big.Int),
		GasLimit:    gasLimit,
		GasPrice:    big.NewInt(0),
	}
	return p
}

func (e Evm) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	var err error
	var usedGas uint64
	defer func(begin time.Time) {
		lvs := []string{"method", "DeliverTx", "error", fmt.Sprint(err != nil)}
		txCount.With(lvs...).Add(1)
		txLatency.With(lvs...).Observe(time.Since(begin).Seconds())
		txGas.With(lvs...).Observe(float64(usedGas))

	}(time.Now())
	origin := common.BytesToAddress(caller.Local)
	vmenv := e.NewEnv(origin)
	runCode, address, leftOverGas, err := vmenv.Create(vm.AccountRef(origin), code, gasLimit, value)
	usedGas = gasLimit - leftOverGas
	loomAddress := loom.Address{
		ChainID: caller.ChainID,
		Local:   address.Bytes(),
	}
	return runCode, loomAddress, err
}

func (e Evm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	var err error
	var usedGas uint64
	defer func(begin time.Time) {
		lvs := []string{"method", "DeliverTx", "error", fmt.Sprint(err != nil)}
		txCount.With(lvs...).Add(1)
		txGas.With(lvs...).Observe(float64(usedGas))
		txLatency.With(lvs...).Observe(time.Since(begin).Seconds())

	}(time.Now())
	origin := common.BytesToAddress(caller.Local)
	contract := common.BytesToAddress(addr.Local)
	vmenv := e.NewEnv(origin)
	ret, leftOverGas, err := vmenv.Call(vm.AccountRef(origin), contract, input, gasLimit, value)
	usedGas = gasLimit - leftOverGas
	return ret, err
}

func (e Evm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	origin := common.BytesToAddress(caller.Local)
	contract := common.BytesToAddress(addr.Local)
	vmenv := e.NewEnv(origin)
	ret, _, err := vmenv.StaticCall(vm.AccountRef(origin), contract, input, gasLimit)
	return ret, err
}

func (e Evm) Commit() (common.Hash, error) {
	root, err := e.state.Commit(true)
	if err == nil {
		e.state.Database().TrieDB().Commit(root, false)
	}
	return root, err
}

func (e Evm) GetCode(addr loom.Address) []byte {
	return e.state.GetCode(common.BytesToAddress(addr.Local))
}

func (e Evm) MintEth(toAddr common.Address, amount *big.Int) {
	e.state.AddBalance(toAddr, amount)
}

func (e Evm) TransferEth(from, to common.Address, amount *big.Int) error {
	if !core.CanTransfer(&e.state, from, amount) {
		return fmt.Errorf("can't transfer %v ETH from %v to %v", amount, from, to)
	}
	core.Transfer(&e.state, from, to, amount)
	return nil
}

func (e Evm) EthBalanceOf(ownerAddr common.Address) *big.Int {
	return e.state.GetBalance(ownerAddr)
}

// TODO: this doesn't need to be exported, rename to newEVM
func (e Evm) NewEnv(origin common.Address) *vm.EVM {
	e.context.Origin = origin
	return vm.NewEVM(e.context, &e.state, &e.chainConfig, e.vmConfig)
}

func defaultChainConfig() params.ChainConfig {
	cliqueCfg := params.CliqueConfig{
		Period: 10,   // Number of seconds between blocks to enforce
		Epoch:  1000, // Epoch length to reset votes and checkpoint
	}
	return params.ChainConfig{
		ChainID:        big.NewInt(0), // Chain id identifies the current chain and is used for replay protection
		HomesteadBlock: nil,           // Homestead switch block (nil = no fork, 0 = already homestead)
		DAOForkBlock:   nil,           // TheDAO hard-fork switch block (nil = no fork)
		DAOForkSupport: true,          // Whether the nodes supports or opposes the DAO hard-fork
		// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
		EIP150Block:         nil,                                  // EIP150 HF block (nil = no fork)
		EIP150Hash:          common.BytesToHash([]byte("myHash")), // EIP150 HF hash (needed for header only clients as only gas pricing changed)
		EIP155Block:         big.NewInt(0),                        // EIP155 HF block
		EIP158Block:         big.NewInt(0),                        // EIP158 HF block
		ByzantiumBlock:      big.NewInt(0),                        // Byzantium switch block (nil = no fork, 0 = already on byzantium)
		ConstantinopleBlock: nil,                                  // Constantinople switch block (nil = no fork, 0 = already activated)
		// Various consensus engines
		Ethash: new(params.EthashConfig),
		Clique: &cliqueCfg,
	}
}

func defaultVmConfig() vm.Config {
	logCfg := vm.LogConfig{
		DisableMemory:  false, // disable memory capture
		DisableStack:   false, // disable stack capture
		DisableStorage: false, // disable storage capture
		Limit:          0,     // maximum length of output, but zero means unlimited
	}
	logger := vm.NewStructLogger(&logCfg)
	return vm.Config{
		// Debug enabled debugging Interpreter options
		Debug: true,
		// Tracer is the op code logger
		Tracer: logger,
		// NoRecursion disabled Interpreter call, callcode,
		// delegate call and create.
		NoRecursion: false,
		// Enable recording of SHA3/keccak preimages
		EnablePreimageRecording: true,
		// JumpTable contains the EVM instruction table. This
		// may be left uninitialised and wille be set to the default
		// table.
		//JumpTable: [256]operation,
	}
}

func defaultContext() vm.Context {
	return vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash: func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		},
		Coinbase:    common.BytesToAddress([]byte("myCoinBase")),
		BlockNumber: new(big.Int),
		Time:        big.NewInt(time.Now().Unix()),
		Difficulty:  new(big.Int),
		GasLimit:    gasLimit,
		GasPrice:    big.NewInt(0),
	}
}

func NewMockEnv(db vm.StateDB, origin common.Address) *vm.EVM {
	chainContext := defaultChainConfig()
	context := defaultContext()
	context.Origin = origin
	return vm.NewEVM(context, db, &chainContext, defaultVmConfig())
}
