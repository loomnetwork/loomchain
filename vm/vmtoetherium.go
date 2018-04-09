package vm

import (
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"math"
	"time"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core/vm"
)

/*
//runtime.NewEnv modified to use vmStateDB rather than state.StateDB
func NewEnv(cfg *runtime.Config, db vm.StateDB) *vm.EVM {
	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },

		Origin:      cfg.Origin,
		Coinbase:    cfg.Coinbase,
		BlockNumber: cfg.BlockNumber,
		Time:        cfg.Time,
		Difficulty:  cfg.Difficulty,
		GasLimit:    cfg.GasLimit,
		GasPrice:    cfg.GasPrice,
	}

	return vm.NewEVM(context, db, cfg.ChainConfig, cfg.EVMConfig)
}

//runtime.Create modified to use vmStateDB rather than state.StateDB
// Create executes the code using the EVM create method
func Create(input []byte, db vm.StateDB) ([]byte, common.Address, uint64, error) {
	//if cfg == nil {
	//	cfg = new(runtime.Config)
	//}
	cfg := getConfig()

	if cfg.State == nil {
		db, _ := ethdb.NewMemDatabase()
		cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(db))
	}
	var (
		vmenv  = NewEnv(&cfg, db)
		sender = vm.AccountRef(cfg.Origin)
	)

	// Call the code with the given configuration.
	code, address, leftOverGas, err := vmenv.Create(
		sender,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return code, address, leftOverGas, err
}

// runtime.Call modified to use vmStateDB rather than state.StateDB
// Call executes the code given by the contract's address. It will return the
// EVM's return value or an error if it failed.
//
// Call, unlike Execute, requires a config and also requires the State field to
// be set.
func Call(address common.Address, input []byte, db vm.StateDB) ([]byte, uint64, error) {
	cfg := getConfig()

	vmenv := NewEnv(&cfg, db)

	//dbstatedb := db.(state.StateDB)
	//sender := dbstatedb.GetOrNewStateObject(cfg.Origin)
	//sender := cfg.State.GetOrNewStateObject(cfg.Origin)
	// Call the code with the given configuration.
	sender := vm.AccountRef(address)

	ret, leftOverGas, err := vmenv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)

	return ret, leftOverGas, err
}
*/
// reimplementation of runtime.setDefaults
// sets defaults on the config
func setDefaults(cfg *runtime.Config)  {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainId:        big.NewInt(1),
			HomesteadBlock: new(big.Int),
			DAOForkBlock:   new(big.Int),
			DAOForkSupport: false,
			EIP150Block:    new(big.Int),
			EIP155Block:    new(big.Int),
			EIP158Block:    new(big.Int),
		}
	}

	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
}

func getConfig() (runtime.Config) {

	cliqueCfg := params.CliqueConfig{
		Period: 10, // Number of seconds between blocks to enforce
		Epoch:  1000,  // Epoch length to reset votes and checkpoint
	}

	chainConfig := params.ChainConfig{
		ChainId: big.NewInt(0),  // Chain id identifies the current chain and is used for replay protection
		HomesteadBlock: nil,  // Homestead switch block (nil = no fork, 0 = already homestead)
		DAOForkBlock:   nil,    // TheDAO hard-fork switch block (nil = no fork)
		DAOForkSupport: true,     // Whether the nodes supports or opposes the DAO hard-fork
		// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
		EIP150Block: nil,     // EIP150 HF block (nil = no fork)
		EIP150Hash: common.StringToHash("myHash"),   // EIP150 HF hash (needed for header only clients as only gas pricing changed)
		EIP155Block: big.NewInt(0), // EIP155 HF block
		EIP158Block: big.NewInt(0),  // EIP158 HF block
		ByzantiumBlock:      nil,      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
		ConstantinopleBlock: nil, // Constantinople switch block (nil = no fork, 0 = already activated)
		// Various consensus engines
		Ethash: new(params.EthashConfig),
		Clique: &cliqueCfg,
	}

	logCfg := vm.LogConfig{
		DisableMemory:  false, // disable memory capture
		DisableStack:   false, // disable stack capture
		DisableStorage: false, // disable storage capture
		Limit:          0,  // maximum length of output, but zero means unlimited
	}
	logger := vm.NewStructLogger(&logCfg)
	evmCfg :=  vm.Config{
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


	cfg := runtime.Config{
		ChainConfig: &chainConfig, // passed to vm.NewEVM
		Difficulty:  big.NewInt(20000), //context
		Origin:      common.StringToAddress("myOrigin"),  //context
		Coinbase:    common.StringToAddress("myCoinBase"),  //context
		BlockNumber: big.NewInt(0),  //context
		Time:        big.NewInt(0),  //context
		GasLimit:    0x2fefd8,  //context
		GasPrice:    big.NewInt(0),  //context
		Value:       big.NewInt(0),  //unused!?
		Debug:       true,  //unused!?
		EVMConfig:   evmCfg, // passed to vm.NewEVM
		//State:     statedb, // passed to vm.NewEVM
		GetHashFn: func(uint64) common.Hash { return common.Hash{} }, //context
	}
	setDefaults(&cfg)
	return cfg
}