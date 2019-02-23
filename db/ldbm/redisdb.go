package ldbm

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var RedisDBBackend = "redisdb"

func init() {
	//	dbCreator := func(url, _ string) (DB, error) {
	//		return NewRedisDB(url)
	//	}

	//TODO expose db hooks from tendermint
	//	registerDBCreator(RedisDBBackend, dbCreator, false)
}

//var _ DB = (*RedisDB)(nil)

type RedisDB struct {
	conn redis.Conn
}

func NewRedisDB(url string) (*RedisDB, error) {
	return NewRedisDBWithOpts(url)
}

func NewRedisDBWithOpts(url string) (*RedisDB, error) {
	fmt.Printf("dialing redis db-%s\n", url)
	c, err := redis.Dial("tcp", "localhost:6379") //url)
	if err != nil {
		return nil, err
	}
	database := &RedisDB{
		conn: c,
	}
	return database, nil
}

// Implements DB.
func (db *RedisDB) Get(key []byte) []byte {
	//key = nonNilBytes(key)
	s, err := redis.Bytes(db.conn.Do("GET", key))
	if err != nil {
		cmn.PanicCrisis(err) //ugh
	}
	return s
}

// Implements DB.
func (db *RedisDB) Has(key []byte) bool {
	exists, err := redis.Bool(db.conn.Do("EXISTS", key))
	if err != nil {
		cmn.PanicCrisis(err) //ugh
	}
	return exists
}

// Implements DB.
func (db *RedisDB) Set(key []byte, value []byte) {
	fmt.Printf("leveldb-set-%s\n", string(key))
	//	key = nonNilBytes(key)
	//	value = nonNilBytes(value)
	_, err := db.conn.Do("SET", key, value)

	if err != nil {
		cmn.PanicCrisis(err)
	}
}

// Implements DB.
func (db *RedisDB) SetSync(key []byte, value []byte) {
	//	key = nonNilBytes(key)
	//	value = nonNilBytes(value)
	_, err := db.conn.Do("SET", key, value) //don't think there is an async vs sync set on redis
	if err != nil {
		cmn.PanicCrisis(err)
	}
}

// Implements DB.
func (db *RedisDB) Delete(key []byte) {
	fmt.Printf("leveldb-Delete-%s\n", string(key))
	//	key = nonNilBytes(key)
	_, err := db.conn.Do("DELETE", key)
	if err != nil {
		cmn.PanicCrisis(err)
	}
}

// Implements DB.
func (db *RedisDB) DeleteSync(key []byte) {
	//	key = nonNilBytes(key)
	_, err := db.conn.Do("DELETE", key) //don't think there is an async vs sync delete on redis
	if err != nil {
		cmn.PanicCrisis(err)
	}
}

// Implements DB.
func (db *RedisDB) Close() {
	db.conn.Close()
}

// Implements DB.
func (db *RedisDB) Print() { /*
		str, _ := db.db.GetProperty("leveldb.stats")
		fmt.Printf("%v\n", str)

		itr := db.db.NewIterator(nil, nil)
		for itr.Next() {
			key := itr.Key()
			value := itr.Value()
			fmt.Printf("[%X]:\t[%X]\n", key, value)
		}
	*/
	panic("not implemented")
}

// Implements DB.
func (db *RedisDB) Stats() map[string]string {
	/*
		keys := []string{
			"leveldb.num-files-at-level{n}",
			"leveldb.stats",
			"leveldb.sstables",
			"leveldb.blockpool",
			"leveldb.cachedblock",
			"leveldb.openedtables",
			"leveldb.alivesnaps",
			"leveldb.aliveiters",
		}
	*/
	stats := make(map[string]string)
	/*
		for _, key := range keys {
			str, err := db.db.GetProperty(key)
			if err == nil {
				stats[key] = str
			}
		}
	*/
	return stats
}

//----------------------------------------
// Batch

//TODO BATCH is not batching!

// Implements DB.
func (db *RedisDB) NewBatch() dbm.Batch {
	return &RedisDBBatch{db}
}

type RedisDBBatch struct {
	db *RedisDB
}

// Implements Batch.
func (mBatch *RedisDBBatch) Set(key, value []byte) {
	//	mBatch.batch.Put(key, value)
	mBatch.db.Set(key, value)
}

// Implements Batch.
func (mBatch *RedisDBBatch) Delete(key []byte) {
	//mBatch.batch.Delete(key)
	mBatch.db.Delete(key)
}

// Implements Batch.
func (mBatch *RedisDBBatch) Write() {
	/*
		err := mBatch.db.db.Write(mBatch.batch, &opt.WriteOptions{Sync: false})
		if err != nil {
			panic(err)
		}
	*/
	//noop cause we aren't batching
}

// Implements Batch.
func (mBatch *RedisDBBatch) WriteSync() {
	/*
		err := mBatch.db.db.Write(mBatch.batch, &opt.WriteOptions{Sync: true})
		if err != nil {
			panic(err)
		}
	*/
	//noop cause we aren't batching
}

//----------------------------------------
// Iterator
// NOTE This is almost identical to db/c_level_db.Iterator
// Before creating a third version, refactor.

// Implements DB.
func (db *RedisDB) Iterator(start, end []byte) dbm.Iterator {
	return newRedisDBIterator(db.conn, start, end, false)
}

// Implements DB.
func (db *RedisDB) ReverseIterator(start, end []byte) dbm.Iterator {
	return newRedisDBIterator(db.conn, start, end, true)
}

type RedisDBIterator struct {
	results     [][]byte
	start       []byte
	end         []byte
	isReverse   bool
	isInvalid   bool
	position    int
	iteration   int
	currentKeys []string
	scan        []interface{}
	conn        redis.Conn
	valid       bool
}

var _ dbm.Iterator = (*RedisDBIterator)(nil)

func newRedisDBIterator(conn redis.Conn, start, end []byte, isReverse bool) *RedisDBIterator {
	//This will return more data then we want
	//redis doesnt have a way to end a range

	keys := []string{}
	iter := 0
	for {
		arr, err := redis.Values(conn.Do("SCAN", iter, "MATCH", fmt.Sprintf("%s*", start)))
		if err != nil {
			cmn.PanicCrisis(err)
			//	return keys, fmt.Errorf("error retrieving '%s' keys", pattern)
		}

		iter, _ = redis.Int(arr[0], nil)
		k, _ := redis.Strings(arr[1], nil)
		fmt.Printf("k is -%v\n", k)
		//		keys = append(keys, k...)
		for _, key := range k {
			//redis won't do range ends, so we have to handle ourselves
			if string(key) < string(end) {
				break
			}
			keys = append(keys, key)
		}

		fmt.Printf("iter is -%v\n", iter)

		if iter == 0 {
			break
		}
	}

	valid := true
	if len(keys) == 0 {
		fmt.Printf("no data for range-%s\n", start)
		//cmn.PanicCrisis("no data") //TODO handle more gracefully?
		valid = false
	}

	if isReverse {
		isReverse = true
	} else {
		/*
			if start == nil {
				source.First()
			} else {
				source.Seek(start)
			}
		*/
		fmt.Printf("Weee scan from front")
	}
	return &RedisDBIterator{
		results:     nil, //results, //TODO for now we preload all data from redis and aren't using a scan
		start:       start,
		end:         end,
		isReverse:   isReverse,
		isInvalid:   false,
		currentKeys: keys,
		position:    0,
		iteration:   iter,
		//	scan:        arr,
		conn:  conn,
		valid: valid,
	}
}

// Implements Iterator.
func (itr *RedisDBIterator) Domain() ([]byte, []byte) {
	return itr.start, itr.end
}

// Implements Iterator.
func (itr *RedisDBIterator) Valid() bool {
	/*

		// Once invalid, forever invalid.
		if itr.isInvalid {
			return false
		}

		// Panic on DB error.  No way to recover.
		itr.assertNoError()

		// If source is invalid, invalid.
		if !itr.source.Valid() {
			itr.isInvalid = true
			return false
		}

		// If key is end or past it, invalid.
		var start = itr.start
		var end = itr.end
		var key = itr.source.Key()

		if itr.isReverse {
			if start != nil && bytes.Compare(key, start) < 0 {
				itr.isInvalid = true
				return false
			}
		} else {
			if end != nil && bytes.Compare(end, key) <= 0 {
				itr.isInvalid = true
				return false
			}
		}
	*/
	// Valid
	return itr.valid
}

// Implements Iterator.
func (itr *RedisDBIterator) Key() []byte {
	// Key returns a copy of the current key.
	itr.assertNoError()
	itr.assertIsValid()
	//return []byte(cp(itr.currentKeys[itr.position]))
	return []byte(itr.currentKeys[itr.position])
}

// Implements Iterator.
func (itr *RedisDBIterator) Value() []byte {
	// Value returns a copy of the current value.
	itr.assertNoError()
	itr.assertIsValid()
	key := itr.currentKeys[itr.position]
	s, err := redis.Bytes(itr.conn.Do("GET", key))
	if err != nil {
		cmn.PanicCrisis(err) //ugh
	}

	return cp(s)
}

///----from tendermint
func cp(bz []byte) (ret []byte) {
	ret = make([]byte, len(bz))
	copy(ret, bz)
	return ret
}

// Implements Iterator.
func (itr *RedisDBIterator) Next() {
	itr.assertNoError()
	itr.assertIsValid()
	if itr.isReverse {
		if itr.iteration == 0 {
			itr.valid = false
		}
		itr.position = itr.position - 1
	} else {

		itr.iteration = itr.iteration + 1
		if len(itr.currentKeys) < itr.iteration {
			itr.valid = false
		}
	}
}

// Implements Iterator.
func (itr *RedisDBIterator) Close() {
	//don't need to do anything cause we don't have any resources open
}

func (itr *RedisDBIterator) assertNoError() {
	/*
		if err := itr.source.Error(); err != nil {
			panic(err)
		}
	*/
	//TODO?
}

func (itr RedisDBIterator) assertIsValid() {
	/*
		if err := itr.source.Error(); err != nil {
			panic(err)
		}
	*/
	//TODO?
}
