// +build evm

package vm

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

	"github.com/loomnetwork/loomchain"
	loom "github.com/loomnetwork/go-loom"
)

var (
	gasLimit = uint64(math.MaxUint64)
	value    = new(big.Int)
)

var EvmFactory = func(state loomchain.State) VM {
	return *NewEvm()
}

type Evm struct {
	state state.StateDB
}

func NewEvm() *Evm {
	p := new(Evm)
	db, _ := ethdb.NewMemDatabase()
	_state, _ := state.New(common.Hash{}, state.NewDatabase(db))
	p.state = *_state
	return p
}

func NewEvmFrom(_state state.StateDB) *Evm {
	p := new(Evm)
	p.state = _state
	return p
}

func (e Evm) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	origin := common.BytesToAddress(caller.Local)
	vmenv := NewEnv(&e.state, origin)
	runCode, address, _, err := vmenv.Create(vm.AccountRef(origin), code, gasLimit, value)
	loomAddress := loom.Address{
		ChainID: caller.ChainID,
		Local:   address.Bytes(),
	}
	return runCode, loomAddress, err
}

func (e Evm) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	origin := common.BytesToAddress(caller.Local)
	contract := common.BytesToAddress(addr.Local)
	vmenv := NewEnv(&e.state, origin)
	ret, _, err := vmenv.Call(vm.AccountRef(origin), contract, input, gasLimit, value)
	return ret, err
}

func (e Evm) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	origin := common.BytesToAddress(caller.Local)
	contract := common.BytesToAddress(addr.Local)
	vmenv := NewEnv(&e.state, origin)
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

func NewEnv(db vm.StateDB, origin common.Address) *vm.EVM {
	cliqueCfg := params.CliqueConfig{
		Period: 10,   // Number of seconds between blocks to enforce
		Epoch:  1000, // Epoch length to reset votes and checkpoint
	}
	chainConfig := params.ChainConfig{
		ChainId:        big.NewInt(0), // Chain id identifies the current chain and is used for replay protection
		HomesteadBlock: nil,           // Homestead switch block (nil = no fork, 0 = already homestead)
		DAOForkBlock:   nil,           // TheDAO hard-fork switch block (nil = no fork)
		DAOForkSupport: true,          // Whether the nodes supports or opposes the DAO hard-fork
		// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
		EIP150Block:         nil,                                  // EIP150 HF block (nil = no fork)
		EIP150Hash:          common.BytesToHash([]byte("myHash")), // EIP150 HF hash (needed for header only clients as only gas pricing changed)
		EIP155Block:         big.NewInt(0),                        // EIP155 HF block
		EIP158Block:         big.NewInt(0),                        // EIP158 HF block
		ByzantiumBlock:      nil,                                  // Byzantium switch block (nil = no fork, 0 = already on byzantium)
		ConstantinopleBlock: nil,                                  // Constantinople switch block (nil = no fork, 0 = already activated)
		// Various consensus engines
		Ethash: new(params.EthashConfig),
		Clique: &cliqueCfg,
	}
	logCfg := vm.LogConfig{
		DisableMemory:  false, // disable memory capture
		DisableStack:   false, // disable stack capture
		DisableStorage: false, // disable storage capture
		Limit:          0,     // maximum length of output, but zero means unlimited
	}
	logger := vm.NewStructLogger(&logCfg)
	evmCfg := vm.Config{
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
	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash: func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		},

		Origin:      origin,
		Coinbase:    common.BytesToAddress([]byte("myCoinBase")),
		BlockNumber: new(big.Int),
		Time:        big.NewInt(time.Now().Unix()),
		Difficulty:  new(big.Int),
		GasLimit:    gasLimit,
		GasPrice:    big.NewInt(0),
	}

	return vm.NewEVM(context, db, &chainConfig, evmCfg)
}
