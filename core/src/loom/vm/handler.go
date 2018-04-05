package vm

import (
	"github.com/gogo/protobuf/proto"
	"loom"
	"loom/store"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/common"
	//"fmt"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/state"
	tmcommon "github.com/tendermint/tmlibs/common"
)

type evmState struct {
	loom.State
	evmDB state.StateDB
}

var vmPrefix = []byte("vm")

func ProcessSendTx(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	vmState := store.PrefixKVStore(state, vmPrefix)
	vmState.Set(tx.To.Local, tx.Code)

	//Send create transaction to EVM
	cfg := getConfig(state.(evmState).evmDB)
	res, _, txErr := runtime.Call(common.BytesToAddress(tx.To.Local), tx.Code, &cfg)

	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	r.Tags = append(r.Tags,kvpResult)
	return r, txErr
}

func ProcessDeployTx(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	// Store EVM byte code
	vmState := store.PrefixKVStore(state, vmPrefix)
	vmState.Set(tx.To.Local, tx.Code)

	//Send create transaction to EVM
	cfg := getConfig(state.(evmState).evmDB)
	res, addr, _, txErr := runtime.Create(tx.Code, &cfg)

	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	kvpAddr := tmcommon.KVPair{[]byte{1}, addr[:]}
	r.Tags = append(r.Tags,kvpResult)
	r.Tags = append(r.Tags,kvpAddr)
	return r, txErr
}

func getConfig(statedb state.StateDB) (runtime.Config) {

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
		// may be left uninitialised and will be set to the default
		// table.
		//JumpTable: [256]operation,
	}

	//db, _ := ethdb.NewMemDatabase()
	//genAllocation := make(core.GenesisAlloc) //empty one is ok?????
	//statedb := tests.MakePreState(db, genAllocation)

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
		State:     &statedb, // passed to vm.NewEVM
		GetHashFn: func(uint64) common.Hash { return common.Hash{} }, //context
	}
	return cfg
}