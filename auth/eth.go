// +build evm

package auth

import (
	"crypto/sha256"

	"github.com/loomnetwork/go-loom/common/evmcompat"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

func verifySolidity66Byte(tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	tronAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, err
	}
	return tronAddr.Bytes(), nil
}

func verifyBinance(tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	hash := sha256.Sum256(tx.Inner)
	addr, err := evmcompat.RecoverAddressFromTypedSig(hash[:], tx.Signature, allowSigTypes)
	if err != nil {
		return nil, err
	}
	return addr.Bytes(), nil
}
