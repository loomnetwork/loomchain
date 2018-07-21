// +build !evm

package keystore

import (
	"fmt"
)

// NewDefaultKeyStore creates an instance of the default key store.
func NewDefaultKeyStore(cfg *DefaultKeyStoreConfig) (KeyStore, error) {
	return &defaultKeyStore{
		store: map[string]keyStoreSignFunc{},
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
