package store

import (
	"log"
	"os"

	"github.com/loomnetwork/go-loom/plugin"
)

type LogParams struct {
	LogFilename    string
	LogFlags       int // log.Ldate | log.Ltime | log.LUTC
	LogVersion     bool
	LogDelete      bool
	LogSetKey      bool
	LogSetValue    bool
	LogSetSize     bool
	LogGet         bool
	LogRange       bool
	LogHas         bool
	LogSaveVersion bool
	LogHash        bool
}

type LogStore struct {
	store  VersionedKVStore
	logger log.Logger
	params LogParams
}

func NewLogStore(store VersionedKVStore) (ls *LogStore, err error) {
	ls = new(LogStore)
	ls.store = store
	ls.params = LogParams{
		LogFilename:    "app-store.log",
		LogFlags:       0,
		LogVersion:     false,
		LogDelete:      true,
		LogSetKey:      true,
		LogSetValue:    false,
		LogSetSize:     true,
		LogGet:         false,
		LogRange:       false,
		LogHas:         false,
		LogSaveVersion: false,
		LogHash:        false,
	}

	file, err := os.Create(ls.params.LogFilename)
	if err != nil {
		return nil, err
	}
	ls.logger = *log.New(file, "", ls.params.LogFlags)
	ls.logger.Println("Created new app log store")
	return ls, nil
}

func (s *LogStore) Delete(key []byte) {
	if s.params.LogDelete {
		s.logger.Println("Delete key: ", string(key))
	}
	s.store.Delete(key)
}

func (s *LogStore) Set(key, val []byte) {
	if s.params.LogSetKey {
		s.logger.Println("Set key: ", string(key))
	}
	if s.params.LogSetValue {
		s.logger.Println("Set Value: ", string(val))
	}
	if s.params.LogSetSize {
		s.logger.Println("Set Size: ", len(val))
	}
	s.store.Set(key, val)
}

func (s *LogStore) Has(key []byte) bool {
	if s.params.LogHas {
		s.logger.Println("Has key: ", string(key))
	}
	return s.store.Has(key)
}

func (s *LogStore) Range(prefix []byte) plugin.RangeData {
	val := s.store.Range(prefix)
	if s.params.LogRange {
		s.logger.Println("Range prefix: ", string(prefix), " val: ", val)
	}
	return val
}

func (s *LogStore) Get(key []byte) []byte {
	val := s.store.Get(key)
	if s.params.LogGet {
		s.logger.Println("Get key: ", string(key), " val: ", val)
	}
	return val
}

func (s *LogStore) Hash() []byte {
	hash := s.store.Hash()
	if s.params.LogHash {
		s.logger.Println("Hash ", hash)
	}
	return hash
}

func (s *LogStore) Version() int64 {
	version := s.store.Version()
	if s.params.LogVersion {
		s.logger.Println("Version ", version)
	}
	return version
}

func (s *LogStore) SaveVersion() ([]byte, int64, error) {
	vByte, vInt, err := s.store.SaveVersion()
	if s.params.LogSaveVersion {
		s.logger.Println("SaveVersion", string(vByte), " int ", vInt, " err ", err)
	}
	return vByte, vInt, err
}

func (s *LogStore) Prune() error {
	return s.store.Prune()
}

func (s *LogStore) GetSnapshot() Snapshot {
	return s.store.GetSnapshot()
}

func (s *LogStore) VersionExists(version int64) bool {
	return s.store.VersionExists(version)
}

func (s *LogStore) RetrieveVersion(version int64) (VersionedKVStore, error) {
	return s.store.RetrieveVersion(version)
}
