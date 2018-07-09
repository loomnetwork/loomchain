package store

import (
	dbm "github.com/tendermint/tmlibs/db"
	"log"
	"os"
)

var (
	LogFilename    = "app-store.log"
	Flags          = 0 // log.Ldate | log.Ltime | log.LUTC
	LogVersion     = false
	LogDelte       = true
	LogSet         = true
	LogGet         = false
	LogHas         = false
	LogSaveVersion = false
	LogHash        = false
)

type LogStore struct {
	store  *IAVLStore
	logger log.Logger
}

func NewLogStore(db dbm.DB) (ls *LogStore, err error) {
	ls = new(LogStore)
	ls.store, err = NewIAVLStore(db)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(LogFilename)
	if err != nil {
		return nil, err
	}
	ls.logger = *log.New(file, "", Flags)
	ls.logger.Println("Created new app log store")
	return ls, nil
}

func (s *LogStore) Delete(key []byte) {
	if LogDelte {
		s.logger.Println("Delete key: ", string(key))
	}
	s.store.Delete(key)
}

func (s *LogStore) Set(key, val []byte) {
	if LogSet {
		s.logger.Println("Set key: ", string(key))
	}
	s.store.Set(key, val)
}

func (s *LogStore) Has(key []byte) bool {
	if LogHas {
		s.logger.Println("Has key: ", string(key))
	}
	return s.store.Has(key)
}

func (s *LogStore) Get(key []byte) []byte {
	val := s.store.Get(key)
	if LogGet {
		s.logger.Println("Get key: ", string(key), " val: ", val)
	}
	return val
}

func (s *LogStore) Hash() []byte {
	hash := s.store.Hash()
	if LogHash {
		s.logger.Println("Hash ", hash)
	}
	return hash
}

func (s *LogStore) Version() int64 {
	version := s.store.Version()
	if LogVersion {
		s.logger.Println("Version ", version)
	}
	return version
}

func (s *LogStore) SaveVersion() ([]byte, int64, error) {
	vByte, vInt, err := s.store.SaveVersion()
	if LogSaveVersion {
		s.logger.Println("SaveVersion", string(vByte), " int ", vInt, " err ", err)
	}
	return vByte, vInt, err
}
