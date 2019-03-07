// +build evm

package auth

import (
	"github.com/loomnetwork/go-ethereum/common"
	"github.com/loomnetwork/go-ethereum/crypto"
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
	ethLocalAdr := common.BytesToAddress(crypto.Keccak256(tx.PublicKey[1:])[12:])
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(crypto.Keccak256(ethLocalAdr.Bytes()), tx.Signature)

	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	return ethAddr.Bytes(), nil
}
