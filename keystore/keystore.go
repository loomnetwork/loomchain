// +build evm

package keystore

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom/common/evmcompat"
)

// MainnetTransferGatewayKeyID should be used to generate a signature in the format expected by
// the Mainnet Transfer Gateway Solidity contract.
const MainnetTransferGatewayKeyID = "mainnet_gateway"

// NewDefaultKeyStore creates an instance of the default key store.
func NewDefaultKeyStore(cfg *DefaultKeyStoreConfig) (KeyStore, error) {
	store := map[string]keyStoreSignFunc{}

	if cfg.MainnetTransferGatewayKey != "" {
		mainnetTransferGatewayKey, err := crypto.LoadECDSA(cfg.MainnetTransferGatewayKey)
		if err != nil {
			return nil, err
		}

		store[MainnetTransferGatewayKeyID] = func(data []byte) ([]byte, error) {
			return signTransferGatewayWithdrawal(data, mainnetTransferGatewayKey)
		}
	}

	return &defaultKeyStore{
		store: store,
	}, nil
}

type keyStoreSignFunc = func(data []byte) ([]byte, error)

// defaultKeyStore implements the KeyStore interface.
type defaultKeyStore struct {
	store map[string]keyStoreSignFunc
}

func (ks *defaultKeyStore) Sign(data []byte, keyID string) ([]byte, error) {
	if sign, exists := ks.store[keyID]; exists {
		return sign(data)
	}
	return nil, fmt.Errorf("[defaultKeyStore] invalid key ID: %s", keyID)
}

func signTransferGatewayWithdrawal(hash []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	sig, err := evmcompat.SoliditySign(hash, key)
	if err != nil {
		return nil, err
	}
	// The first byte should be the signature mode, for details about the signature format refer to
	// https://github.com/loomnetwork/plasma-erc721/blob/master/server/contracts/Libraries/ECVerify.sol
	return append(make([]byte, 1, 66), sig...), nil
}
