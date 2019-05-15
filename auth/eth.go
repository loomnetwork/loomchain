// +build evm

package auth

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
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
	hash := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(tx.Inner))))
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash[:], tx.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieve public key from eos signature %v", tx.Signature)
	}
	return ethAddr.Bytes(), nil
}

func LocalAddressFromEosPublicKey(eccPublicKey ecc.PublicKey) (loom.LocalAddress, error) {
	btcecPubKey, err := eccPublicKey.Key()
	if err != nil {
		return nil, errors.Wrapf(err, "retrieve btcec key from eos key %v", eccPublicKey)
	}
	return loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ecdsa.PublicKey(*btcecPubKey)).Hex())
}
