package keystore

// KeyStore provides the ability to sign with one of the Node's private keys.
type KeyStore interface {
	// Sign signs the given data using the key & algo associated with the given keyID.
	// Returns the signature, or an error.
	Sign(data []byte, keyID string) ([]byte, error)
}

type DefaultKeyStoreConfig struct {
	// Path to file containing a hex-encoded secp256k1 private key.
	MainnetTransferGatewayKey string
}
