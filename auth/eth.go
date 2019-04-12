// +build evm

package auth

import (
	"crypto/ecdsa"

	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/ethereum/go-ethereum/crypto"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
)

func verifySolidity66Byte(tx SignedTx) ([]byte, error) {
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(tx SignedTx) ([]byte, error) {
	publicKeyBytes, err := crypto.Ecrecover(sha3.SoliditySHA3(tx.Inner), tx.Signature)
	if err != nil {
		return nil, err
	}
	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	return crypto.PubkeyToAddress(*publicKey).Bytes(), nil
}

func verifyEos(tx SignedTx) ([]byte, error) {
	signature := ecc.NewSigNil()
	if _, err := signature.Unpack(tx.Signature); err != nil {
		return nil, errors.Wrapf(err, "unpack eos signature %v", tx.Signature)
	}
	eosPubKey, err := signature.PublicKey(sha3.SoliditySHA3(tx.Inner))
	if err != nil {
		return nil, errors.Wrapf(err, "retrieve public key from eos signature %v", tx.Signature)
	}
	return LocalAddressFromEosPublicKey(eosPubKey)
}


func LocalAddressFromEosPublicKey(eccPublicKey ecc.PublicKey) (loom.LocalAddress, error) {
	btcecPubKey, err := eccPublicKey.Key()
	if err != nil {
		return nil, errors.Wrapf(err, "retrieve btcec key from eos key %v", eccPublicKey)
	}
	return loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ecdsa.PublicKey(*btcecPubKey)).Hex())
}