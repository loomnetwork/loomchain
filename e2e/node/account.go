package node

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	"github.com/binance-chain/go-sdk/keys"
	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	loom "github.com/loomnetwork/go-loom"
)

type Account struct {
	PubKey      string
	PubKeyPath  string
	PrivKey     string
	PrivKeyPath string
	Address     string
	Local       string
}

func CreateAccount(id int, baseDir, loompath string) (*Account, error) {
	pubfile := path.Join(baseDir, fmt.Sprintf("pubkey-%d", id))
	privfile := path.Join(baseDir, fmt.Sprintf("privkey-%d", id))
	out, err := exec.Command(loompath, "genkey", "-a", pubfile, "-k", privfile).Output()
	if err != nil {
		return nil, err
	}
	pubKey, err := ioutil.ReadFile(pubfile)
	if err != nil {
		return nil, err
	}
	privKey, err := ioutil.ReadFile(privfile)
	if err != nil {
		return nil, err
	}
	acct := &Account{
		PubKey:      string(pubKey),
		PubKeyPath:  pubfile,
		PrivKey:     string(privKey),
		PrivKeyPath: privfile,
	}
	for _, s := range strings.Split(string(out), "\n") {
		if i := strings.Index(s, "local address: "); i > -1 {
			acct.Address = strings.TrimPrefix(s[i:], "local address: ")
		}
		if i := strings.Index(s, "local address base64: "); i > -1 {
			acct.Local = strings.TrimPrefix(s[i:], "local address base64: ")
		}
	}
	if acct.Address == "" || acct.Local == "" {
		return nil, errors.New("address must not be blank")
	}
	return acct, nil
}

type EthAccount struct {
	PubKey      string
	PubKeyPath  string
	PrivKey     *ecdsa.PrivateKey
	PrivKeyPath string
	Address     string
	Local       string
}

type TronAccount struct {
	PubKey      string
	PubKeyPath  string
	PrivKey     *ecdsa.PrivateKey
	PrivKeyPath string
	Address     string
	Local       string
}

type BinanceAccount struct {
	PubKey      string
	PubKeyPath  string
	PrivKey     *ecdsa.PrivateKey
	PrivKeyPath string
	Address     string
	Local       string
}

func CreateEthAccount(id int, baseDir string) (*EthAccount, error) {
	ethKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	privfile := path.Join(baseDir, fmt.Sprintf("privethkey-%d", id))
	if err := crypto.SaveECDSA(privfile, ethKey); err != nil {
		return nil, err
	}

	local, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
	if err != nil {
		return nil, err
	}
	addr := loom.Address{ChainID: "default", Local: local}
	return &EthAccount{
		Address:     addr.String(),
		Local:       local.String(),
		PrivKey:     ethKey,
		PrivKeyPath: privfile,
	}, nil
}

func CreateTronAccount(id int, baseDir string) (*TronAccount, error) {
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}
	tronKey := key.ToECDSA()

	privfile := path.Join(baseDir, fmt.Sprintf("privetronkey-%d", id))
	if err := crypto.SaveECDSA(privfile, tronKey); err != nil {
		return nil, err
	}

	local, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(tronKey.PublicKey).Hex())
	if err != nil {
		return nil, err
	}
	addr := loom.Address{ChainID: "tron", Local: local}
	return &TronAccount{
		Address:     addr.String(),
		Local:       local.String(),
		PrivKey:     tronKey,
		PrivKeyPath: privfile,
	}, nil
}

func CreateBinanceAccount(id int, baseDir string) (*BinanceAccount, error) {
	keyManager, err := keys.NewKeyManager()
	if err != nil {
		return nil, err
	}
	hexPrivKey, err := keyManager.ExportAsPrivateKey()
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.HexToECDSA(hexPrivKey)
	if err != nil {
		return nil, err
	}

	privfile := path.Join(baseDir, fmt.Sprintf("privebinancekey-%d", id))
	if err := crypto.SaveECDSA(privfile, privKey); err != nil {
		return nil, err
	}

	local, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privKey.PublicKey).Hex())
	if err != nil {
		return nil, err
	}
	addr := loom.Address{ChainID: "binance", Local: local}
	return &BinanceAccount{
		Address:     addr.String(),
		Local:       local.String(),
		PrivKey:     privKey,
		PrivKeyPath: privfile,
	}, nil
}
