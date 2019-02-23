package db

import "github.com/loomnetwork/loomchain/db/ldbm"

type RedisDB struct {
	*ldbm.RedisDB
}

func (g *RedisDB) Compact() error {
	panic("unsupported!")
	return nil
	//	return g.DB().CompactRange(util.Range{})
}

func LoadRedisDB(name, dir string) (*RedisDB, error) {
	db, err := ldbm.NewRedisDB(name)
	if err != nil {
		return nil, err
	}

	return &RedisDB{RedisDB: db}, nil
}
