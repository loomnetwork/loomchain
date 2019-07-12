package lib

import (
	"bytes"
	"io/ioutil"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/e2e/node"
)

type Config struct {
	Name         string
	BaseDir      string
	LoomPath     string
	ContractDir  string
	Nodes        map[string]*node.Node
	Accounts     []*node.Account
	EthAccounts  []*node.EthAccount
	TronAccounts []*node.TronAccount
	TestFile     string
	LogAppDb     bool
	// helper to easy access by template
	AccountAddressList     []string
	AccountPrivKeyPathList []string
	AccountPubKeyList      []string

	EthAccountAddressList     []string
	EthAccountPrivKeyPathList []string
	EthAccountPubKeyList      []string

	TronAccountAddressList     []string
	TronAccountPrivKeyPathList []string
	TronAccountPubKeyList      []string

	NodeAddressList         []string
	NodeBase64AddressList   []string
	NodePubKeyList          []string
	NodePrivKeyPathList     []string
	NodeRPCAddressList      []string
	NodeProxyAppAddressList []string

	AlwasyApphashCheck      bool
}

func WriteConfig(conf Config, filename string) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(conf); err != nil {
		return errors.Wrapf(err, "encoding runner config error")
	}

	configPath := path.Join(conf.BaseDir, filename)
	if err := ioutil.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func ReadConfig(filename string) (Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}
