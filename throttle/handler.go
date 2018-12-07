package throttle

import (
	"github.com/loomnetwork/go-loom"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/pkg/errors"
	lauth "github.com/loomnetwork/loomchain/auth"
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

func (dv *OriginValidator) ValidateOrigin(txBytes []byte, chainId string) error {
	if !dv.deployValidation && !dv.callValidation {
		return nil
	}

	var txSigned auth.SignedTx
	if err := proto.Unmarshal(txBytes, &txSigned); err != nil  {
		return  err
	}
	origin, err := lauth.GetOrigin(txSigned, chainId)
	if err != nil {
		return err
	}

	var txNonce auth.NonceTx
	if err := proto.Unmarshal(txSigned.Inner, &txNonce); err != nil {
		return err
	}

	var txTransaction types.Transaction
	if err := proto.Unmarshal(txNonce.Inner, &txTransaction); err!= nil  {
		return err
	}

	var txMessage vm.MessageTx
	if err := proto.Unmarshal(txTransaction.Data, &txMessage); err != nil {
		return err
	}

	switch txTransaction.Id {
	case callId: return dv.validateCaller(origin)
	case deployId:return dv.validateDeployer(origin)
	default: return errors.Errorf("unrecognised transaction id %v", txTransaction.Id)
	}
}

func (dv *OriginValidator) validateDeployer(deployer loom.Address) error {
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

func (dv *OriginValidator) validateCaller(caller loom.Address) error {
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
