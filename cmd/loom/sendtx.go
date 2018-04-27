package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loom/client"
	"github.com/loomnetwork/loom/vm"
)

type deployTxFlags struct {
	Bytecode   string `json:"bytecode"`
	PublicFile string `json:"publicfile"`
	PrivFile   string `json:"privfile"`
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

	rpcclient := client.NewDAppChainRPCClient("tcp://localhost", 46657, 9999)
	respB, err := rpcclient.CommitDeployTx(clientAddr, signer, loom.VMType_EVM, bytecode)

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
	return callCmd

}

func callTx(addr, input, privFile, publicFile string) ([]byte, error) {

	contractAddrS, err := decodeHexString(addr)
	if err != nil {
		return nil, err
	}
	contractAddr := loom.Address{
		ChainID: "default",
		Local:   loom.LocalAddress(contractAddrS),
	}

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

	rpcclient := client.NewDAppChainRPCClient("tcp://localhost", 46657, 9999)
	resp, err := rpcclient.CommitCallTx(clientAddr, contractAddr, signer, loom.VMType_EVM, incode)
	if err != nil {
		return nil, err
	}

	response := types.CallResponse{}
	err = proto.Unmarshal(resp, &response)
	return response.Output, err
}

func caller(privFile, publicFile string) (loom.Address, loom.Signer, error) {
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
		ChainID: "default",
		Local:   localAddr,
	}
	signer := loom.NewEd25519Signer(privKey)
	return clientAddr, signer, err
}
