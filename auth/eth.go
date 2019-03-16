// +build evm

package auth

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/pkg/errors"
)

func verifySolidity66Byte(tx SignedTx) ([]byte, error) {
	hash, err := auth.GetTxHash(tx.Inner)
	if err != nil {
		return nil, errors.Wrapf(err, "get hash from tx %v", tx)
	}

	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(tx SignedTx) ([]byte, error) {
	hash, err := auth.GetTxHash(tx.Inner)
	if err != nil {
		return nil, errors.Wrapf(err, "get hash from tx %v", tx)
	}
	publicKeyBytes, err := crypto.Ecrecover(hash, tx.Signature)
	if err != nil {
		return nil, err
	}
	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes);
	if err != nil {
		return nil, err
	}
	ethAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(*publicKey).Hex())
	if err != nil {
		return nil, err
	}

	return ethAddr, nil
}
