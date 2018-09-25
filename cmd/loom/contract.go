package main

import (
	`encoding/base64`
	"github.com/gogo/protobuf/proto"
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/auth`
	`github.com/loomnetwork/go-loom/cli`
	`github.com/loomnetwork/go-loom/client`
	"github.com/pkg/errors"
	`github.com/spf13/cobra`
	`io/ioutil`
)

var (
	contractTxFlags struct {
		PublicFile   string `json:"publicfile"`
		PrivFile     string `json:"privfile"`
	}
)

func newContractCmd(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "call a method of the " +  name + " contract",
	}
	pflags := cmd.PersistentFlags()
	pflags.StringVarP(&contractTxFlags.PublicFile, "address", "a", "", "address file")
	pflags.StringVarP(&contractTxFlags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(pflags)
	return cmd
}

func contract(name string) (*client.Contract, error) {
	rpcClient := client.NewDAppChainRPCClient(testChainFlags.ChainID, testChainFlags.WriteURI, testChainFlags.ReadURI)
	contractAddr, err := rpcClient.Resolve(name)
	if err != nil {
		return nil, errors.Wrap(err, "resolve address")
	}
	contract := client.NewContract(rpcClient, contractAddr.Local)
	return contract, nil
}

func callContract(name string, method string, params proto.Message, result interface{}) error {
	if contractTxFlags.PrivFile == "" {
		return errors.New("private key required to call contract")
	}
	
	privKeyB64, err := ioutil.ReadFile(contractTxFlags.PrivFile)
	if err != nil {
		return errors.Wrap(err, "read private key file")
	}
	
	privKey, err := base64.StdEncoding.DecodeString(string(privKeyB64))
	if err != nil {
		return errors.Wrap(err, "private key decode")
	}
	
	signer := auth.NewEd25519Signer(privKey)
	
	contract, err := contract(name)
	if err != nil {
		return err
	}
	_, err = contract.Call(method, params, signer, result)
	return err
}

func staticCallContract(name string, method string, params proto.Message, result interface{}) error {
	contract, err := contract(name)
	if err != nil {
		return errors.Wrapf(err, "get contract %s",name)
	}
	
	_, err = contract.StaticCall(method, params, loom.RootAddress(testChainFlags.ChainID), result)
	return err
}

func resolveAddress(s string) (loom.Address, error) {
	rpcClient := client.NewDAppChainRPCClient(testChainFlags.ChainID, testChainFlags.WriteURI, testChainFlags.ReadURI)
	contractAddr, err := cli.ParseAddress(s)
	if err != nil {
		contractAddr, err = rpcClient.Resolve(s)
		if err != nil {
			return loom.Address{}, errors.Wrap(err, "resolve address")
		}
	}
	return contractAddr, nil
}