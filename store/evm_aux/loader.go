package evmaux

import (
	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

func LoadEvmAuxStore() (*EvmAuxStore, error) {
	evmAuxDB, err := goleveldb.OpenFile(EvmAuxDBName, nil)
	if err != nil {
		return nil, err
	}
	return NewEvmAuxStore(evmAuxDB), nil
}
