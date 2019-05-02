// +build evm

package evm

import (
	"log"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/loomnetwork/go-loom"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	multiWriterLogger        log.Logger
	multiWriterLoggerStarted = false
)

const (
	EVM_DB      = 1
	LOOM_ETH_DB = 2
	ALL_DB      = 3
)

type MultiWriterDB interface {
	Delete([]byte) error
	Put([]byte, []byte) error
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
	Close()
	NewBatch() ethdb.Batch
	WithLogContext(*MultiWriterDBLogContext) MultiWriterDB
}

type MultiWriterDBConfig struct {
	Read  int
	Write int
}

var _ MultiWriterDB = &multiWriterDB{}

type multiWriterDB struct {
	evmDB     dbm.DB
	loomEthDB *LoomEthDB
	sync.Mutex
	lock       sync.RWMutex
	logContext *MultiWriterDBLogContext
	config     MultiWriterDBConfig
}

type MultiWriterDBLogContext struct {
	BlockHeight  int64
	ContractAddr loom.Address
	CallerAddr   loom.Address
}

func NewMultiWriterDB(evmDB dbm.DB, loomEthDB *LoomEthDB, logContext *MultiWriterDBLogContext, config MultiWriterDBConfig) MultiWriterDB {
	return &multiWriterDB{
		evmDB:      evmDB,
		loomEthDB:  loomEthDB,
		logContext: logContext,
		config:     config,
	}
}

func (db *multiWriterDB) Delete(key []byte) error {
	if db.config.Write == EVM_DB || db.config.Write == ALL_DB {
		db.evmDB.Delete(key)
	}
	if db.config.Write == LOOM_ETH_DB || db.config.Write == ALL_DB {
		db.loomEthDB.Delete(key)
	}
	return nil
}

func (db *multiWriterDB) Put(key []byte, value []byte) error {
	if db.config.Write == EVM_DB || db.config.Write == ALL_DB {
		db.evmDB.Set(key, value)
	}
	if db.config.Write == LOOM_ETH_DB || db.config.Write == ALL_DB {
		db.loomEthDB.Put(key, value)
	}
	return nil
}

func (db *multiWriterDB) Get(key []byte) ([]byte, error) {
	if db.config.Read == EVM_DB {
		return db.evmDB.Get(key), nil
	} else {
		return db.loomEthDB.Get(key)
	}
}

func (db *multiWriterDB) Has(key []byte) (bool, error) {
	if db.config.Read == EVM_DB {
		return db.evmDB.Has(key), nil
	} else {
		return db.loomEthDB.Has(key)
	}
}

func (db *multiWriterDB) Close() {
}

func (db *multiWriterDB) NewBatch() ethdb.Batch {
	return db.NewLogBatch(db.logContext)
}

func (db *multiWriterDB) WithLogContext(logContext *MultiWriterDBLogContext) MultiWriterDB {
	return &multiWriterDB{
		evmDB:      db.evmDB,
		loomEthDB:  db.loomEthDB,
		logContext: logContext,
		config:     db.config,
	}
}

func (db *multiWriterDB) NewLogBatch(logContext *MultiWriterDBLogContext) ethdb.Batch {
	b := new(multiWriterLogBatch)
	b.batch = *new(multiWriterBatch)
	b.batch.db = db
	b.batch.Reset()
	b.params = MultiWriterLogParams{
		LogFilename:        "multiwriter-batch.log",
		LogFlags:           0,
		LogReset:           true,
		LogDelete:          true,
		LogWrite:           true,
		LogValueSize:       false,
		LogPutKey:          true,
		LogPutValue:        false,
		LogPutDump:         false,
		LogWriteDump:       true,
		LogBeforeWriteDump: false,
	}

	if !multiWriterLoggerStarted {
		file, err := os.Create(b.params.LogFilename)
		if err != nil {
			return &b.batch
		}
		multiWriterLogger = *log.New(file, "", b.params.LogFlags)
		multiWriterLogger.Println("Created ethdb batch logger")
		multiWriterLoggerStarted = true
	}

	if logContext != nil {
		multiWriterLogger.Printf(
			batchHeaderWithContext,
			logContext.BlockHeight,
			logContext.ContractAddr,
			logContext.CallerAddr,
		)
	} else {
		multiWriterLogger.Print(batchHeader)
	}
	return b
}

// implements ethdb.Batch
type multiWriterKVPair struct {
	key   []byte
	value []byte
}

type multiWriterBatch struct {
	cache []multiWriterKVPair
	db    *multiWriterDB
	size  int
}

func (b *multiWriterBatch) Put(key, value []byte) error {
	b.cache = append(b.cache, multiWriterKVPair{
		key:   common.CopyBytes(key),
		value: common.CopyBytes(value),
	})
	b.size += len(value)
	return nil
}

func (b *multiWriterBatch) ValueSize() int {
	return b.size
}

func (b *multiWriterBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.cache {
		if kv.value == nil {
			b.db.Delete(kv.key)
		} else {
			b.db.Put(kv.key, kv.value)
		}
	}
	return nil
}

func (b *multiWriterBatch) Reset() {
	b.cache = make([]multiWriterKVPair, 0)
	b.size = 0
}

func (b *multiWriterBatch) Delete(key []byte) error {
	b.cache = append(b.cache, multiWriterKVPair{
		key:   common.CopyBytes(key),
		value: nil,
	})
	return nil
}

func (b *multiWriterBatch) Dump(multiLogger *log.Logger) {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()
	multiLogger.Print("\n---- BATCH DUMP ----\n")
	for i, kv := range b.cache {
		multiLogger.Printf("IDX %d, KEY %s\n", i, kv.key)
	}
}

type MultiWriterLogParams struct {
	LogFilename        string
	LogFlags           int
	LogReset           bool
	LogDelete          bool
	LogWrite           bool
	LogValueSize       bool
	LogPutKey          bool
	LogPutValue        bool
	LogPutDump         bool
	LogWriteDump       bool
	LogBeforeWriteDump bool
}

type multiWriterLogBatch struct {
	batch  multiWriterBatch
	params MultiWriterLogParams
}

const headerWithContext = `

-----------------------------
| NEW BATCH
| Block: %v
| Contract: %v
| Caller: %v
-----------------------------

`

const header = `

-----------------------------
| NEW BATCH
-----------------------------

`

func (b *multiWriterLogBatch) Delete(key []byte) error {
	if b.params.LogDelete {
		multiWriterLogger.Println("Delete key: ", string(key))
	}
	return b.batch.Delete(key)
}

func (b *multiWriterLogBatch) Put(key, value []byte) error {
	if b.params.LogPutKey {
		multiWriterLogger.Println("Put key: ", string(key))
	}
	if b.params.LogPutValue {
		multiWriterLogger.Println("Put value: ", string(value))
	}
	err := b.batch.Put(key, value)
	if b.params.LogPutDump {
		b.batch.Dump(&multiWriterLogger)
	}
	return err
}

func (b *multiWriterLogBatch) ValueSize() int {
	size := b.batch.ValueSize()
	if b.params.LogValueSize {
		multiWriterLogger.Println("ValueSize : ", size)
	}
	return size
}

func (b *multiWriterLogBatch) Write() error {
	if b.params.LogWrite {
		multiWriterLogger.Println("Write")
	}
	if b.params.LogBeforeWriteDump {
		multiWriterLogger.Println("Write, before : ")
		b.batch.Dump(&multiWriterLogger)
	}
	err := b.batch.Write()
	if b.params.LogWriteDump {
		multiWriterLogger.Println("Write, after : ")
		b.batch.Dump(&multiWriterLogger)
	}
	return err
}

func (b *multiWriterLogBatch) Reset() {
	if b.params.LogReset {
		multiWriterLogger.Println("Reset batch")
	}
	b.batch.Reset()
}
