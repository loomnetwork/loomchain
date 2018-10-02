package plasma_cash

import (
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client/plasma_cash/eth"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/oracle"
)

type EthClientSerializableConfig struct {
	// URI of an Ethereum node
	EthereumURI string
	// Plasma contract address on Ethereum
	PlasmaHexAddress string
	// Path of Private key that should be used to sign txs sent to Ethereum
	PrivateKeyPath string
	// Override default gas computation when sending txs to Ethereum
	OverrideGas bool
	// How often Ethereum should be polled for mined txs
	TxPollInterval int64
	// Maximum amount of time to way for a tx to be mined by Ethereum
	TxTimeout int64
}

type DAppChainSerializableConfig struct {
	WriteURI string
	ReadURI  string
	// Used to sign txs sent to Loom DAppChain
	PrivateKeyPath string
}

type OracleSerializableConfig struct {
	PlasmaBlockInterval uint32
	DAppChainCfg        *DAppChainSerializableConfig
	EthClientCfg        *EthClientSerializableConfig
}

type PlasmaCashSerializableConfig struct {
	OracleEnabled   bool
	ContractEnabled bool

	OracleConfig *OracleSerializableConfig
}

type PlasmaCashConfig struct {
	OracleEnabled   bool
	ContractEnabled bool

	OracleConfig *oracle.OracleConfig
}

func DefaultConfig() *PlasmaCashSerializableConfig {
	// Default config disables oracle, so
	// no need to populate oracle config
	// with default vaule.
	return &PlasmaCashSerializableConfig{
		OracleEnabled:   false,
		ContractEnabled: false,
	}
}

func loadDAppChainPrivateKey(path string) ([]byte, error) {
	privKeyB64, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privKey, err := base64.StdEncoding.DecodeString(string(privKeyB64))
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

func loadMainnetPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	privKey, err := crypto.LoadECDSA(path)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func ParseSerializableConfig(chainID string, serializableConfig *PlasmaCashSerializableConfig) (*PlasmaCashConfig, error) {
	plasmaCashConfig := &PlasmaCashConfig{
		ContractEnabled: serializableConfig.ContractEnabled,
		OracleEnabled:   serializableConfig.OracleEnabled,
	}

	if !serializableConfig.OracleEnabled {
		return plasmaCashConfig, nil
	}

	if serializableConfig.OracleConfig == nil {
		return nil, fmt.Errorf("plasma cash oracle config is missing")
	}

	if serializableConfig.OracleConfig.DAppChainCfg == nil {
		return nil, fmt.Errorf("plasma cash oracle's DAppchain config is missing")
	}

	fmt.Println(serializableConfig.OracleConfig.DAppChainCfg)

	if serializableConfig.OracleConfig.EthClientCfg == nil {
		return nil, fmt.Errorf("plasma cash oracle's etherum config is missing")
	}

	dAppChainPrivateKey, err := loadDAppChainPrivateKey(serializableConfig.OracleConfig.DAppChainCfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	mainnetPrivateKey, err := loadMainnetPrivateKey(serializableConfig.OracleConfig.EthClientCfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	plasmaCashConfig.OracleConfig = &oracle.OracleConfig{
		DAppChainClientCfg: oracle.DAppChainPlasmaClientConfig{
			ChainID:  chainID,
			WriteURI: serializableConfig.OracleConfig.DAppChainCfg.WriteURI,
			ReadURI:  serializableConfig.OracleConfig.DAppChainCfg.ReadURI,
			Signer:   auth.NewEd25519Signer(dAppChainPrivateKey),
		},
		EthClientCfg: eth.EthPlasmaClientConfig{
			EthereumURI:      serializableConfig.OracleConfig.EthClientCfg.EthereumURI,
			PlasmaHexAddress: serializableConfig.OracleConfig.EthClientCfg.PlasmaHexAddress,
			PrivateKey:       mainnetPrivateKey,
			OverrideGas:      serializableConfig.OracleConfig.EthClientCfg.OverrideGas,
			TxPollInterval:   time.Duration(serializableConfig.OracleConfig.EthClientCfg.TxPollInterval),
			TxTimeout:        time.Duration(serializableConfig.OracleConfig.EthClientCfg.TxTimeout),
		},
	}

	return plasmaCashConfig, nil

}
