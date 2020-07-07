// +build evm

package config

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

type PlasmaCashConfig struct {
	OracleEnabled   bool
	ContractEnabled bool

	StatusServiceAddress string

	OracleConfig *oracle.OracleConfig
}

func LoadSerializableConfig(chainID string, serializableConfig *PlasmaCashSerializableConfig) (*PlasmaCashConfig, error) {
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
		return nil, fmt.Errorf("plasma cash oracle's Loom Protocol config is missing")
	}

	if serializableConfig.OracleConfig.EthClientCfg == nil {
		return nil, fmt.Errorf("plasma cash oracle's Ethereum config is missing")
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
		PlasmaBlockInterval:  serializableConfig.OracleConfig.PlasmaBlockInterval,
		StatusServiceAddress: serializableConfig.OracleConfig.StatusServiceAddress,
		DAppChainClientCfg: oracle.DAppChainPlasmaClientConfig{
			ChainID:      chainID,
			WriteURI:     serializableConfig.OracleConfig.DAppChainCfg.WriteURI,
			ReadURI:      serializableConfig.OracleConfig.DAppChainCfg.ReadURI,
			Signer:       auth.NewEd25519Signer(dAppChainPrivateKey),
			ContractName: serializableConfig.OracleConfig.DAppChainCfg.ContractName,
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
