package vm

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/common"
	tmcommon "github.com/tendermint/tmlibs/common"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/core/state"
	"fmt"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"github.com/ethereum/go-ethereum/params"
	"github.com/loomnetwork/loom/auth"
)

var (
	contextKeySender = "sender"
	//var vmPrefix = []byte("vm")
	//var rootKey = []byte{}
	rootKey = []byte("vmroot")
)


func ProcessSendTx(loomState loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult


	tx := &SendTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}
	fmt.Println("address to", tx.Address, " code ", tx.Input)
	// Store EVM byte code
	//vmState := store.PrefixKVStore(state, vmPrefix)
	//vmState.Set(tx.To.Local, tx.Code)

	evmStore :=  NewEvmStore(loomState)
	cfg := getConfig(*evmStore)

	res, _, err := runtime.Call(common.BytesToAddress(tx.Address), tx.Input, &cfg)
	kvpResult := tmcommon.KVPair{[]byte{0}, res}

	root, _ := cfg.State.Commit(true)
	evmStore.Put(rootKey,  root[:])
	cfg.State.Database().TrieDB().Commit(root, false)

	r.Tags = append(r.Tags,kvpResult)
	return r, err
}

func ProcessDeployTx(loomState loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult //Tags []common.KVPair

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}
	fmt.Println( " code ", tx.Input)

	// Store EVM byte code
	//vmState := store.PrefixKVStore(state, vmPrefix)
	//vmState.Set(tx.To.Local, tx.Code)

	evmStore :=  NewEvmStore(loomState)
	cfg := getConfig(*evmStore)

	res, addr, _, err := runtime.Create(tx.Input, &cfg)
	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	kvpAddr := tmcommon.KVPair{[]byte{1}, addr[:]}

	root, _ := cfg.State.Commit(true)
	evmStore.Put(rootKey,  root[:])
	cfg.State.Database().TrieDB().Commit(root, false)

	r.Tags = append(r.Tags,kvpResult)
	r.Tags = append(r.Tags,kvpAddr)
	return r, err
}

func getConfig(evmDB evmStore) (runtime.Config) {

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
		//Origin:      common.StringToAddress("myOrigin"),  //context
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

	oldRoot, _ := evmDB.Get(rootKey)
	cfg.State, _ = state.New(common.BytesToHash(oldRoot), state.NewDatabase(&evmDB))

	if nil != evmDB.state.Context().Value(contextKeySender) {
		sender := auth.Sender(evmDB.state.Context())
		cfg.Origin = common.StringToAddress(sender.String())
	} else {
		cfg.Origin = common.StringToAddress("myOrigin")
	}

	return cfg
}