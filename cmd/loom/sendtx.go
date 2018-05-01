package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
)

type deployTxFlags struct {
	Bytecode   string `json:"bytecode"`
	PublicFile string `json:"publicfile"`
	PrivFile   string `json:"privfile"`
}

var writeURI, readURI, chainID string

func setChainFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&writeURI, "write", "w", "http://localhost:46657", "URI for sending txs")
	fs.StringVarP(&readURI, "read", "r", "http://localhost:47000", "URI for quering app state")
	fs.StringVarP(&chainID, "chain", "", "default", "chain ID")
}

func newDeployCommand() *cobra.Command {
	var flags deployTxFlags
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, runBytecode, err := deployTx(flags.Bytecode, flags.PrivFile, flags.PublicFile)
			if err != nil {
				return err
			}
			fmt.Println("New contract deployed with address: ", addr)
			fmt.Println("Runtime bytecode: ", runBytecode)
			return nil
		},
	}
	deployCmd.Flags().StringVarP(&flags.Bytecode, "bytecode", "b", "", "bytecode file")
	deployCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	deployCmd.Flags().StringVarP(&flags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(deployCmd.Flags())
	return deployCmd
}

func deployTx(bcFile, privFile, pubFile string) (loom.Address, []byte, error) {
	clientAddr, signer, err := caller(privFile, pubFile)
	if err != nil {
		return *new(loom.Address), nil, err
	}

	bytetext, err := ioutil.ReadFile(bcFile)
	if err != nil {
		return *new(loom.Address), nil, err
	}
	if string(bytetext[0:2]) == "0x" {
		bytetext = bytetext[2:]
	}
	bytecode, err := hex.DecodeString(string(bytetext))
	if err != nil {
		return *new(loom.Address), nil, err
	}

	rpcclient := client.NewDAppChainRPCClient(chainID, writeURI, readURI)
	respB, err := rpcclient.CommitDeployTx(clientAddr, signer, vm.VMType_EVM, bytecode)
	if err != nil {
		return *new(loom.Address), nil, err
	}
	response := vm.DeployResponse{}
	err = proto.Unmarshal(respB, &response)
	if err != nil {
		return *new(loom.Address), nil, err
	}
	addr := loom.UnmarshalAddressPB(response.Contract)

	return addr, response.Output, nil
}

type callTxFlags struct {
	ContractAddr string `json:"contractaddr"`
	Input        string `json:"input"`
	PublicFile   string `json:"publicfile"`
	PrivFile     string `json:"privfile"`
}

func newCallCommand() *cobra.Command {
	var flags callTxFlags

	callCmd := &cobra.Command{
		Use:   "call",
		Short: "Call a contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := callTx(flags.ContractAddr, flags.Input, flags.PrivFile, flags.PublicFile)
			if err != nil {
				return err
			}
			fmt.Println("Call response: ", resp)
			return nil
		},
	}
	callCmd.Flags().StringVarP(&flags.ContractAddr, "contract-addr", "c", "", "contract address")
	callCmd.Flags().StringVarP(&flags.Input, "input", "i", "", "file with input data")
	callCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	callCmd.Flags().StringVarP(&flags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(callCmd.Flags())
	return callCmd

}

func callTx(addr, input, privFile, publicFile string) ([]byte, error) {
	contractAddr := AddressFromString(addr)

	clientAddr, signer, err := caller(privFile, publicFile)
	if err != nil {
		return nil, err
	}

	intext, err := ioutil.ReadFile(input)
	if err != nil {
		return nil, err
	}
	if string(intext[0:2]) == "0x" {
		intext = intext[2:]
	}
	incode, err := hex.DecodeString(string(intext))
	if err != nil {
		return nil, err
	}

	rpcclient := client.NewDAppChainRPCClient(chainID, writeURI, readURI)
	resp, err := rpcclient.CommitCallTx(clientAddr, contractAddr, signer, vm.VMType_EVM, incode)
	if err != nil {
		return nil, err
	}

	response := types.CallResponse{}
	err = proto.Unmarshal(resp, &response)
	return response.Output, err
}

func caller(privFile, publicFile string) (loom.Address, auth.Signer, error) {
	privKey, err := ioutil.ReadFile(privFile)
	if err != nil {
		log.Fatalf("Cannot read priv key: %s", privFile)
	}
	addr, err := ioutil.ReadFile(publicFile)
	if err != nil {
		log.Fatalf("Cannot read address file: %s", publicFile)
	}
	localAddr := loom.LocalAddressFromPublicKey(addr)
	log.Println(localAddr)
	clientAddr := loom.Address{
		ChainID: chainID,
		Local:   localAddr,
	}
	signer := auth.NewEd25519Signer(privKey)
	return clientAddr, signer, err
}

func AddressFromString(addr string) loom.Address {
	indexColumn := strings.Index(addr, ":")
	if indexColumn < 0 || indexColumn+2 > len(addr) {
		return loom.Address{}
	}
	local, err := decodeHexString(addr[indexColumn+1:])
	if err != nil {
		return loom.Address{}
	}
	return loom.Address{
		ChainID: addr[0 : indexColumn+1],
		Local:   loom.LocalAddress(local),
	}
}
