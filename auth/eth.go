// +build evm

package auth

import (
	"github.com/loomnetwork/go-loom/common/evmcompat"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

func VerifySolidity66Byte(tx SignedTx) ([]byte, error) {
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(tx SignedTx) ([]byte, error) {
	tronAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature)
	if err != nil {
		return nil, err
	}
	return tronAddr.Bytes(), nil
}
