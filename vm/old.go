package vm

//While running the ethereum virtual machine(EVM) the current state of the machine is stored in a StateDB.
// This StateDB contains three data structures.
//1) State Objects: This contains the live objects used during normal running of the EVM.
// A map keyed by address. There is also a map of flags indicating if any of the state objects are dirty,
// or different from the data in the Trie.
//
//2) Trie: This is a Merkle Patricia tree of objects flattened by rlp. This is used for historical data.
// When a piece of data is not in the State Object map, it is sought here then copied over to the live map.
//
//3) Database: An ethdb that the Trie data can be read to and written from.
// It is used by go-ethereum for sequencing database during writing to a blockchain.
// In our used case it a loom.State object that can be written directly to a blockchain.
//
//We are given a loom.State object and wish the state of the EVM to be written to it. To do this we do the following.
//1) Wrap the loom.State object in an LoomEthdb object, this implements the ethdb interface
// so that it can be used as the backing database for a go-ethereum StateDB.
// LoomEthdb :=  NewLoomEthdb(loomState)
//
//2) Create a new state.StateDB using our LoomEthdb. We also need to provide the root of the Trie we want
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
// LoomEthdb.Put(rootKey,  root[:])
//
//6) Call commit on the Trie. This copies the data in the Tire to the loom.State database.
// The root from above is used to indicate that all the Tire tree is to be copied.
// cfg.State.Database().TrieDB().Commit(root, false)
//
//7) All the information needed to run further transactions are now in the loom.State, and the other
// objects can be forgotten.
/*
var (
	contextKeySender = "sender"
	eventKey = []byte("events")
)

func ProcessSendTx(loomState loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	tx := &SendTx{}
	err := proto.Unmarshal(txBytes, tx)
	if err != nil {
		return r, err
	}

	LoomEthdb :=  NewLoomEthdb(loomState)
	cfg := getConfig()
	oldRoot, _ := LoomEthdb.Get(rootKey)
	cfg.State, _ = state.New(common.BytesToHash(oldRoot), state.NewDatabase(LoomEthdb))
	if nil != LoomEthdb.ctx.Value(contextKeySender) {
		sender := auth.Origin(LoomEthdb.ctx)
		cfg.Origin = common.StringToAddress(sender.String())
	} else {
		cfg.Origin = common.StringToAddress("myOrigin")
	}

	res, _, err := runtime.Call(common.BytesToAddress(tx.Address), tx.Input, &cfg)
	kvpResult := tmcommon.KVPair{[]byte{0}, res}

	root, _ := cfg.State.Commit(true)
	LoomEthdb.Put(rootKey,  root[:])
	cfg.State.Database().TrieDB().Commit(root, false)
	handleEvents(*LoomEthdb, cfg.State.Logs())

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

	LoomEthdb :=  NewLoomEthdb(loomState)
	cfg := getConfig()

	oldRoot, _ := LoomEthdb.Get(rootKey)
	cfg.State, _ = state.New(common.BytesToHash(oldRoot), state.NewDatabase(LoomEthdb))
	if nil != LoomEthdb.ctx.Value(contextKeySender) {
		sender := auth.Origin(LoomEthdb.ctx)
		cfg.Origin = common.StringToAddress(sender.String())
	} else {
		cfg.Origin = common.StringToAddress("myOrigin")
	}

	res, addr, _, err := runtime.Create(tx.Input, &cfg)
	kvpResult := tmcommon.KVPair{[]byte{0}, res}
	kvpAddr := tmcommon.KVPair{[]byte{1}, addr[:]}

	root, _ := cfg.State.Commit(true)
	LoomEthdb.Put(rootKey,  root[:])
	cfg.State.Database().TrieDB().Commit(root, false)
	handleEvents(*LoomEthdb, cfg.State.Logs())

	r.Tags = append(r.Tags,kvpResult)
	r.Tags = append(r.Tags,kvpAddr)
	return r, err
}

func handleEvents(evmDB LoomEthdb, logs []*types.Log) {
	var events []*Event
	for _,v := range logs {
		var topics [][]byte
		for value := range v.Topics {
			topics = append(topics, v.Topics[value].Bytes())
		}
		events = append(events, &Event{
			v.Address.Bytes(),
			topics,
			v.Data,
		})
	}
	protoEvnets := &Events{
		events,
	}
	eventsB, err := proto.Marshal(protoEvnets)
	if err != nil {
		fmt.Println("error marshaling events", err)
	}
	oldEvents, _ := evmDB.Get(eventKey)
	newEvents := append(oldEvents, eventsB...)
	evmDB.Put(eventKey, newEvents)
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
		Difficulty:   new(big.Int), //context
		//Origin:      common.StringToAddress("myOrigin"),  //context
		Coinbase:    common.StringToAddress("myCoinBase"),  //context
		BlockNumber: nil,  //context
		Time:        big.NewInt(time.Now().Unix()),  //context
		GasLimit:    0,  //context
		GasPrice:    nil,  //context
		Value:       nil,  //unused!?
		Debug:       true,  //unused!?
		EVMConfig:   evmCfg, // passed to vm.NewEVM
		//State:     statedb, // passed to vm.NewEVM
		GetHashFn: 	func(n uint64) common.Hash {
						return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
					}, //context
	}



	return cfg
}*/