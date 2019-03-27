//+build !evm

package gateway

import (
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain/fnConsensus"
	"github.com/pkg/errors"
)

type BatchSignWithdrawalFn struct {
}

func (b *BatchSignWithdrawalFn) PrepareContext() (bool, []byte, error) {
	return false, nil, nil
}

func (b *BatchSignWithdrawalFn) SubmitMultiSignedMessage(ctx []byte, key []byte, signatures [][]byte) {

}

func (b *BatchSignWithdrawalFn) GetMessageAndSignature(ctx []byte) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (b *BatchSignWithdrawalFn) MapMessage(ctx, key, message []byte) error {
	return nil
}

func CreateBatchSignWithdrawalFn(isLoomcoinFn bool, chainID string, fnRegistry fnConsensus.FnRegistry, tgConfig *TransferGatewayConfig, signer auth.Signer) (*BatchSignWithdrawalFn, error) {
	return nil, errors.New("not implemented in non-EVM build")
}
