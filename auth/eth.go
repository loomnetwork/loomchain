// +build evm

package auth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/pkg/errors"
)

func verifySolidity65Byte(tx SignedTx) ([]byte, error) {
	ethAddr, err := evmcompat.SolidityRecover(tx.PublicKey, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	return ethAddr.Bytes(), nil
}

func verifySolidity66Byte(tx SignedTx) ([]byte, error) {
	hash := crypto.Keccak256(common.BytesToAddress(crypto.Keccak256(tx.PublicKey[1:])[12:]).Bytes())

	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	return ethAddr.Bytes(), nil
}

func verifySolidity66Byte2(tx SignedTx) ([]byte, error) {
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(tx.PublicKey, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	return ethAddr.Bytes(), nil
}