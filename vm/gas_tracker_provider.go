package vm

import (
	"errors"
	"math"
	"math/big"

	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
)

type GasTrackerProvider struct {
	createCoinCtx func(state loomchain.State) (contract.Context, error)
	tracker       loomchain.GasTracker
}

func NewGasTrackerProvider(createCoinCtx func(state loomchain.State) (contract.Context, error)) *GasTrackerProvider {
	return &GasTrackerProvider{
		createCoinCtx: createCoinCtx,
	}
}

func (p *GasTrackerProvider) CreateTracker(state loomchain.State) (loomchain.GasTracker, error) {
	var gv uint32
	txHandlerCfg := state.Config().GetTxHandler()
	gv = txHandlerCfg.GetGasTrackerVersion()

	maxGas := state.Config().GetEvm().GetGasLimit()
	if maxGas == 0 {
		maxGas = math.MaxUint64
	}

	switch gv {
	case 0:
		p.tracker = NewLegacyGasTracker(maxGas)
	case 1:
		coinCtx, err := p.createCoinCtx(state)
		if err != nil {
			return nil, err
		}
		var feeCollector loom.Address
		if txHandlerCfg.GetFeeCollector() != nil {
			feeCollector = loom.UnmarshalAddressPB(txHandlerCfg.GetFeeCollector())
		}
		var minPrice *big.Int
		if txHandlerCfg.GetMinGasPrice() != nil {
			minPrice = txHandlerCfg.GetMinGasPrice().Value.Int
		}
		p.tracker = NewLoomCoinGasTracker(coinCtx, feeCollector, maxGas, minPrice)
	default:
		return nil, errors.New("invalid gas tracker version")
	}
	return p.tracker, nil
}

func (p *GasTrackerProvider) GetTracker() loomchain.GasTracker {
	return p.tracker
}
