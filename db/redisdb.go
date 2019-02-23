package db

import dbm "github.com/tendermint/tendermint/libs/db"

type RedisDB struct {
	*dbm.RedisDB
}

func (g *RedisDB) Compact() error {
	panic("unsupported!")
	return nil
	//	return g.DB().CompactRange(util.Range{})
}

func LoadRedisDB(name, dir string) (*RedisDB, error) {
	db, err := dbm.NewRedisDB(name)
	if err != nil {
		return nil, err
	}

	return &RedisDB{RedisDB: db}, nil
}
