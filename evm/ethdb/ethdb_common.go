package ethdb

type EthDbType int

const (
	EthDbNone EthDbType = iota
	EthDbLoom
	EthDbLdbDatabase
	EthDbMemDb
)
