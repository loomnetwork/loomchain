// +build !evm

package oracle

import "errors"

type OracleConfig struct {
}

// Not implemented in non-EVM build
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

// PlasmaCoinWorker sends Plasma deposits from Ethereum to Loom Protocol.
type PlasmaCoinWorker struct {
}

func NewPlasmaCoinWorker(cfg *OracleConfig) *PlasmaCoinWorker {
	return nil
}

func (w *PlasmaCoinWorker) Init() error {
	return errors.New("not implemented in non-EVM build")
}

func (w *PlasmaCoinWorker) Run() {
}

type Oracle struct {
	cfg         *OracleConfig
	coinWorker  *PlasmaCoinWorker
	blockWorker *PlasmaBlockWorker
}

func NewOracle(cfg *OracleConfig) *Oracle {
	return &Oracle{
		cfg:         cfg,
		coinWorker:  NewPlasmaCoinWorker(cfg),
		blockWorker: NewPlasmaBlockWorker(cfg),
	}
}

func (orc *Oracle) Init() error {
	return errors.New("not implemented in non-EVM build")
}

func (orc *Oracle) Run() {

}
