package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/types"
	lauth "github.com/loomnetwork/loomchain/auth"
	"github.com/pkg/errors"
)

type callTx struct {
	origin loom.Address
	nonce  uint64
}

type TxLimiterConfig struct {
	LimitDeploys        bool
	LimitCalls          bool
	DeployerAddressList []string
	CallSessionDuration int64
}

func DefaultTxLimiterConfig() *TxLimiterConfig {
	return &TxLimiterConfig{
		LimitDeploys:        false,
		LimitCalls:          false,
		CallSessionDuration: 1,
	}
}

func (c *TxLimiterConfig) DeployerAddresses() ([]loom.Address, error) {

	deployerAddressList := make([]loom.Address, 0, len(c.DeployerAddressList))
	for _, addrStr := range c.DeployerAddressList {
		addr, err := loom.ParseAddress(addrStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing deploy address %s", addrStr)
		}
		deployerAddressList = append(deployerAddressList, addr)
	}
	return deployerAddressList, nil
}

// Clone returns a deep clone of the config.
func (c *TxLimiterConfig) Clone() *TxLimiterConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

type OriginValidator struct {
	period           uint64
	alreadyCalled    [][]callTx
	allowedDeployers []loom.Address
	deployValidation bool
	callValidation   bool
}

func NewOriginValidator(period uint64, allowedDeployers []loom.Address, deployValidation, callValidation bool) OriginValidator {
	dv := OriginValidator{
		period:           period,
		alreadyCalled:    make([][]callTx, period),
		allowedDeployers: allowedDeployers,
		deployValidation: deployValidation,
		callValidation:   callValidation,
	}
	return dv
}

func (dv *OriginValidator) ValidateOrigin(txBytes []byte, chainId string, currentBlockHeight int64) error {
	if !dv.deployValidation && !dv.callValidation {
		return nil
	}

	var txSigned auth.SignedTx
	if err := proto.Unmarshal(txBytes, &txSigned); err != nil {
		return err
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
	if err := proto.Unmarshal(txNonce.Inner, &txTransaction); err != nil {
		return err
	}

	switch txTransaction.Id {
	case callId:
		return dv.validateCaller(origin, txNonce.Sequence, uint64(currentBlockHeight))
	case deployId:
		return dv.validateDeployer(origin)
	default:
		return errors.Errorf("unrecognised transaction id %v", txTransaction.Id)
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

func (dv *OriginValidator) validateCaller(caller loom.Address, nonce, currentBlockHeight uint64) error {
	if !dv.callValidation {
		return nil
	}
	for _, callersBlock := range dv.alreadyCalled {
		for _, called := range callersBlock {
			if 0 == caller.Compare(called.origin) && nonce != called.nonce {
				return errors.Errorf("already placed call tx; try again in %v blocks", dv.period)
			}
		}
	}
	callerBlockIndex := int(currentBlockHeight) % int(dv.period)
	dv.alreadyCalled[callerBlockIndex] = append(dv.alreadyCalled[callerBlockIndex], callTx{caller, nonce})
	return nil
}

func (dv *OriginValidator) Reset(currentBlockHeight int64) {
	callerBlockIndex := int(currentBlockHeight) % int(dv.period)
	dv.alreadyCalled[callerBlockIndex] = []callTx{{}}
}
