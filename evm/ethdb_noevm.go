// +build !evm

package evm

type EthDbManager interface {}

type EthDbHandler struct {}

func NewEthDbHandler(ethDbType EthDbType) EthDbManager {
	if ethDbType != EthDbNone {
		panic("Only EthDb type None allowed in non evm build")
	}
	return EthDbHandler{}
}