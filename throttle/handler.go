package throttle

import (
	"github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
)

type OriginValidator struct {
	period              uint64
	alreadyCalled       []loom.Address
	allowedDeployers    []loom.Address
	deployValidation    bool
	callValidation      bool
}

func NewOrginHandler(period uint64, allowedDeployers []loom.Address, deployValidation, callValidation bool) OriginValidator {
	dv := OriginValidator{
		period:             period,
		alreadyCalled:      nil,
		allowedDeployers:   allowedDeployers,
		deployValidation:   deployValidation,
		callValidation:     callValidation,
	}
	return dv
}

func (dv *OriginValidator) ValidateDeployer(deployer loom.Address) error {
	if !dv.deployValidation {
		return nil
	}
	for _, allowed := range dv.allowedDeployers {
		if 0 == deployer.Compare(allowed) {
			return nil
		}
	}
	return errors.Errorf("origin not on list of users registered for deploys")
}

func (dv *OriginValidator) ValidateCaller(caller loom.Address) error {
	if !dv.callValidation {
		return nil
	}
	for _, called := range dv.alreadyCalled {
		if 0 == caller.Compare(called) {
			return errors.Errorf("already placed call tx; try again in %v blocks", dv.period)
		}
	}
	dv.alreadyCalled = append(dv.alreadyCalled, caller)
	return nil
}


func (dv *OriginValidator) Reset(blockNumber int64) {
	if uint64(blockNumber) % dv.period == 0 {
		dv.alreadyCalled = nil
	}
}