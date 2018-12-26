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

	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2/oracle"
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

func LoadSerializableConfig(chainID string, oracleConfig *OracleSerializableConfig) (*oracle.Config, error) {
	cfg := &oracle.Config{
		Enabled: false,
	}

	if oracleConfig == nil || !oracleConfig.Enabled {
		return cfg, nil
	}

	if oracleConfig.DAppChainCfg == nil {
		return nil, fmt.Errorf("dposv2 oracle's DAppchain config is missing")
	}

	if oracleConfig.EthClientCfg == nil {
		return nil, fmt.Errorf("dposv2 oracle's etherum config is missing")
	}

	dAppChainPrivateKey, err := loadDAppChainPrivateKey(oracleConfig.DAppChainCfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	mainnetPrivateKey, err := loadMainnetPrivateKey(oracleConfig.EthClientCfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	cfg.Enabled = oracleConfig.Enabled
	cfg.StatusServiceAddress = oracleConfig.StatusServiceAddress
	cfg.DAppChainClientCfg = oracle.DAppChainDPOSv2ClientConfig{
		ChainID:      chainID,
		WriteURI:     oracleConfig.DAppChainCfg.WriteURI,
		ReadURI:      oracleConfig.DAppChainCfg.ReadURI,
		Signer:       auth.NewEd25519Signer(dAppChainPrivateKey),
		ContractName: oracleConfig.DAppChainCfg.ContractName,
	}
	cfg.EthClientCfg = oracle.EthClientConfig{
		EthereumURI: oracleConfig.EthClientCfg.EthereumURI,
		PrivateKey:  mainnetPrivateKey,
	}
	cfg.MainnetPollInterval = time.Duration(oracleConfig.MainnetPollInterval)

	// Parsing workers config
	if oracleConfig.TimeLockWorkerCfg == nil {
		cfg.TimeLockWorkerCfg = oracle.TimeLockWorkerConfig{
			Enabled: false,
		}
	} else {
		cfg.TimeLockWorkerCfg = oracle.TimeLockWorkerConfig{
			Enabled:                   oracleConfig.TimeLockWorkerCfg.Enabled,
			TimeLockFactoryHexAddress: oracleConfig.TimeLockWorkerCfg.TimeLockFactoryHexAddress,
		}
	}

	return cfg, nil

}
