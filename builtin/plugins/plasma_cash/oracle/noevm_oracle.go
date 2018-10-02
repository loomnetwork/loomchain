// +build !evm

package oracle

import "errors"

type OracleConfig struct {
}

// Not implemented in non-evm build
type PlasmaBlockWorker struct {
}

func NewPlasmaBlockWorker(cfg *OracleConfig) *PlasmaBlockWorker {
	return nil
}

func (w *PlasmaBlockWorker) Init() error {
	return errors.New("not implemented in non-EVM build")
}

func (w *PlasmaBlockWorker) Run() {

}

// PlasmaDepositWorker sends Plasma deposits from Ethereum to the DAppChain.
type PlasmaDepositWorker struct {
}

func NewPlasmaDepositWorker(cfg *OracleConfig) *PlasmaDepositWorker {
	return nil
}

func (w *PlasmaDepositWorker) Init() error {
	return errors.New("not implemented in non-EVM build")
}

func (w *PlasmaDepositWorker) Run() {
}

type Oracle struct {
	cfg           *OracleConfig
	depositWorker *PlasmaDepositWorker
	blockWorker   *PlasmaBlockWorker
}

func NewOracle(cfg *OracleConfig) *Oracle {
	return &Oracle{
		cfg:           cfg,
		depositWorker: NewPlasmaDepositWorker(cfg),
		blockWorker:   NewPlasmaBlockWorker(cfg),
	}
}

func (orc *Oracle) Init() error {
	return errors.New("not implemented in non-EVM build")
}

// TODO: Graceful shutdown
func (orc *Oracle) Run() {

}
