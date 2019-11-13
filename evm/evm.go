// +build evm

package evm

import (
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/features"
)

// EVMEnabled indicates whether or not Loom EVM integration is available
const (
	EVMEnabled      = true
	defaultGasLimit = math.MaxUint64
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
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "evmtx_latency_microseconds",
		Help:       "Total duration of go-ethereum EVM tx in microseconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)
	txGas = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  "loomchain",
		Subsystem:  "application",
		Name:       "evm_tx_gas_cost",
		Help:       "Gas cost of EVM transaction.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)
}

type evmAccountBalanceManager struct {
	abm     AccountBalanceManager
	chainID string
}

func newEVMAccountBalanceManager(abm AccountBalanceManager, chainID string) *evmAccountBalanceManager {
	return &evmAccountBalanceManager{
		abm:     abm,
		chainID: chainID,
	}
}

func (m *evmAccountBalanceManager) GetBalance(account common.Address) *big.Int {
	addr := loom.Address{
		ChainID: m.chainID,
		Local:   account.Bytes(),
	}
	if balance, err := m.abm.GetBalance(addr); err == nil {
		return balance.Int
	}
	return common.Big0
}

func (m *evmAccountBalanceManager) AddBalance(account common.Address, amount *big.Int) error {
	addr := loom.Address{
		ChainID: m.chainID,
		Local:   account.Bytes(),
	}
	return m.abm.AddBalance(addr, loom.NewBigUInt(amount))
}

func (m *evmAccountBalanceManager) SubBalance(account common.Address, amount *big.Int) error {
	addr := loom.Address{
		ChainID: m.chainID,
		Local:   account.Bytes(),
	}
	return m.abm.SubBalance(addr, loom.NewBigUInt(amount))
}

func (m *evmAccountBalanceManager) SetBalance(account common.Address, amount *big.Int) error {
	addr := loom.Address{
		ChainID: m.chainID,
		Local:   account.Bytes(),
	}
	return m.abm.SetBalance(addr, loom.NewBigUInt(amount))
}

func (m *evmAccountBalanceManager) CanTransfer(from common.Address, amount *big.Int) bool {
	addr := loom.Address{
		ChainID: m.chainID,
		Local:   from.Bytes(),
	}
	if balance, err := m.abm.GetBalance(addr); err == nil {
		return balance.Int.Cmp(amount) >= 0
	}
	return false
}

func (m *evmAccountBalanceManager) Transfer(from, to common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	fromAddr := loom.Address{
		ChainID: m.chainID,
		Local:   from.Bytes(),
	}
	toAddr := loom.Address{
		ChainID: m.chainID,
		Local:   to.Bytes(),
	}
	if fromAddr.Compare(toAddr) == 0 {
		return
	}
	m.abm.Transfer(fromAddr, toAddr, loom.NewBigUInt(amount))
}

type wrappedEVM struct {
	sdb             vm.StateDB
	context         vm.Context
	chainConfig     params.ChainConfig
	vmConfig        vm.Config
	validateTxValue bool
	gasLimit        uint64
}

func NewEvm(
	sdb vm.StateDB, lstate loomchain.State, abm *evmAccountBalanceManager, tracer vm.Tracer,
) (*wrappedEVM, error) {
	p := new(wrappedEVM)
	p.sdb = sdb
	p.gasLimit = lstate.Config().GetEvm().GetGasLimit()
	if p.gasLimit == 0 {
		p.gasLimit = defaultGasLimit
	}

	p.chainConfig = defaultChainConfig(lstate.FeatureEnabled(features.EvmConstantinopleFeature, false))

	var err error
	p.vmConfig, err = createVmConfig(tracer)
	if err != nil {
		return nil, errors.Wrap(err, "creating vm.Config")
	}
	p.validateTxValue = lstate.FeatureEnabled(features.CheckTxValueFeature, false)
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
		GasLimit:    p.gasLimit,
		GasPrice:    big.NewInt(0),
	}
	if abm != nil {
		p.context.CanTransfer = func(db vm.StateDB, addr common.Address, amount *big.Int) bool {
			return abm.CanTransfer(addr, amount)
		}
		p.context.Transfer = func(db vm.StateDB, from, to common.Address, amount *big.Int) {
			abm.Transfer(from, to, amount)
		}
	}
	return p, nil
}

func (e wrappedEVM) Create(caller loom.Address, code []byte, value *loom.BigUInt) ([]byte, loom.Address, error) {
	var err error
	var usedGas uint64
	defer func(begin time.Time) {
		lvs := []string{"method", "DeliverTx", "error", fmt.Sprint(err != nil)}
		txCount.With(lvs...).Add(1)
		txLatency.With(lvs...).Observe(time.Since(begin).Seconds())
		txGas.With(lvs...).Observe(float64(usedGas))

	}(time.Now())
	origin := common.BytesToAddress(caller.Local)
	vmenv := e.newEnv(origin)

	var val *big.Int
	if value == nil {
		val = common.Big0
	} else {
		val = value.Int
		if e.validateTxValue && val.Cmp(common.Big0) < 0 {
			return nil, loom.Address{}, errors.Errorf("value %v must be non negative", value)
		}
	}

	runCode, address, leftOverGas, err := vmenv.Create(vm.AccountRef(origin), code, e.gasLimit, val)
	usedGas = e.gasLimit - leftOverGas
	loomAddress := loom.Address{
		ChainID: caller.ChainID,
		Local:   address.Bytes(),
	}
	return runCode, loomAddress, err
}

func (e wrappedEVM) Call(caller, addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
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
	vmenv := e.newEnv(origin)

	var val *big.Int
	if value == nil {
		val = common.Big0
	} else {
		val = value.Int
		if val == nil {
			//there seems like there are serialization issues where we can get bad data here
			val = common.Big0
		}
		if e.validateTxValue && val.Cmp(common.Big0) < 0 {
			return nil, errors.Errorf("value %v must be non negative", value)
		}
	}
	ret, leftOverGas, err := vmenv.Call(vm.AccountRef(origin), contract, input, e.gasLimit, val)
	usedGas = e.gasLimit - leftOverGas
	return ret, err
}

func (e wrappedEVM) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	origin := common.BytesToAddress(caller.Local)
	contract := common.BytesToAddress(addr.Local)
	vmenv := e.newEnv(origin)
	ret, _, err := vmenv.StaticCall(vm.AccountRef(origin), contract, input, e.gasLimit)
	return ret, err
}

func (e wrappedEVM) GetCode(addr loom.Address) []byte {
	return e.sdb.GetCode(common.BytesToAddress(addr.Local))
}

func (e wrappedEVM) GetStorageAt(addr loom.Address, key []byte) ([]byte, error) {
	result := e.sdb.GetState(common.BytesToAddress(addr.Local), common.BytesToHash(key))
	return result.Bytes(), nil
}

func (e wrappedEVM) newEnv(origin common.Address) *vm.EVM {
	e.context.Origin = origin
	return vm.NewEVM(e.context, e.sdb, &e.chainConfig, e.vmConfig)
}

func defaultChainConfig(enableConstantinople bool) params.ChainConfig {
	cliqueCfg := params.CliqueConfig{
		Period: 10,   // Number of seconds between blocks to enforce
		Epoch:  1000, // Epoch length to reset votes and checkpoint
	}

	var constantinopleBlock *big.Int
	if enableConstantinople {
		constantinopleBlock = big.NewInt(0)
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
		ConstantinopleBlock: constantinopleBlock,                  // Constantinople switch block (nil = no fork, 0 = already activated)
		// Various consensus engines
		Ethash: new(params.EthashConfig),
		Clique: &cliqueCfg,
	}
}

func createVmConfig(tracer vm.Tracer) (vm.Config, error) {
	if tracer == nil {
		logCfg := vm.LogConfig{
			DisableMemory:  true, // disable memory capture
			DisableStack:   true, // disable stack capture
			DisableStorage: true, // disable storage capture
			Limit:          0,    // maximum length of output, but zero means unlimited
		}
		tracer = vm.NewStructLogger(&logCfg)
	}

	return vm.Config{
		// Debug enabled debugging Interpreter options
		Debug: tracer != nil,
		// Tracer is the op code logger
		Tracer: tracer,
		// NoRecursion disabled Interpreter call, callcode,
		// delegate call and create.
		NoRecursion: false,
		// Enable recording of SHA3/keccak preimages
		EnablePreimageRecording: true, //TODO: make this optional, [MGC] I don't think we need to keep this
		// JumpTable contains the EVM instruction table. This
		// may be left uninitialised and will be set to the default
		// table.
		//JumpTable: [256]operation,
	}, nil
}
