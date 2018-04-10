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

//While running the ethereum virtual machine(EVM) the current state of the machine is stored in a StateDB.
// This StateDB contains three data structures.
//1) State Objects: This contains the live objects used during normal running of the EVM.
// A map keyed by address. There is also a map of flags indicating if any of the state objects are dirty,
// or different from the data in the Trie.
//
//2) Trie: This is a Merkle Patricia tree of objects flattened by rlp. This is used for historical data.
// When a piece is of data is not in the State Object map, it is sought here then copied over to the live map.
//
//3) Database: An ethdb that the Trie data can be read to and written from.
// It is used by go-ethereum for sequencing database during writing to a blockchain.
// In our used case it a loom.State object that can be written directly to a blockchain.
//
//We are given a loom.State object and wish the state of the EVM to be written to it. To do this we do the following.
//1) Wrap the loom.State object in an evmStore object, this implements the ethdb interface
// so can be used as the backing database for a go-ethereum StateDB.
// evmStore :=  NewEvmStore(loomState)
//
//2) Create a new state.StateDB using our evmStore. We also need to provide the root of the Trie we want
// to be read from the database. Initially, this will be zeroed to indicate a new database,
// but in the future incarnation, the Trie root will need to be remembered from last time.
// oldRoot, _ := evmDB.Get(rootKey)
// cfg.State, _ = state.New(common.BytesToHash(oldRoot), state.NewDatabase(&evmDB))
//
//3) Enter the new StateDB with the config information (cfg) when running Create and Call EVM functions.
// res, _, err := runtime.Call(common.BytesToAddress(tx.Address), tx.Input, &cfg)
// res, addr, _, err := runtime.Create(tx.Input, &cfg)
//
//4) On finishing, in preparation for writing to the blockchain, we do the following.
//
//5) Call commit on the StateDB. This copies the data from the live cache into the Tire.
// It also returns the root of the Tire, this needs to be remembered for the next time the current state is used.
// root, _ := cfg.State.Commit(true)
// evmStore.Put(rootKey,  root[:])
//
//6) Call commit on the Trie. This copies the data in the Tire to the loom.State database.
// The root from above is used to indicate that all the Tire tree is to be copied.
// cfg.State.Database().TrieDB().Commit(root, false)
//
//7) All the information needed to run further transactions are now in the loom.State, and the other
// objects can be forgotten.

var (
	contextKeySender = "sender"
	rootKey = []byte("root")
)

func ProcessSendTx(loomState loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	tx := &SendTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}
	fmt.Println("address to", tx.Address, " code ", tx.Input)

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
	var r loom.TxHandlerResult

	tx := &DeployTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}
	fmt.Println( " code ", tx.Input)

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

	if nil != evmDB.ctx.Value(contextKeySender) {
		sender := auth.Sender(evmDB.ctx)
		cfg.Origin = common.StringToAddress(sender.String())
	} else {
		cfg.Origin = common.StringToAddress("myOrigin")
	}

	return cfg
}