// +build evm

package throttle

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
)

func isEthDeploy(txBytes []byte) (bool, error) {
	var tx types.Transaction
	if err := rlp.DecodeBytes(txBytes, &tx); err != nil {
		return false, errors.Wrap(err, "decoding ethereum transaction")
	}
	return tx.To() == nil, nil
}
