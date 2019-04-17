// +build !evm

package ethdb

type EthDbManager interface {}

type EthDbHandler struct {}

func NewEthDbManager(ethDbType EthDbType) EthDbManager {
	if ethDbType != EthDbNone {
		panic("Only EthDb type None allowed in non evm build")
	}
	return EthDbHandler{}
}