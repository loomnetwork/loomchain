package evm

type EthDbType int

const (
	EthDbNone EthDbType = iota
	EthDbLoom
	EthDbLdbDatabase
	// EthDbMemDb
)
