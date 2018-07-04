package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/vm"
)

type deployTxFlags struct {
	Bytecode   string `json:"bytecode"`
	PublicFile string `json:"publicfile"`
	PrivFile   string `json:"privfile"`
	Name       string `json:"name"`
}

type chainFlags struct {
	WriteURI string
	ReadURI  string
	ChainID  string
}

var testChainFlags chainFlags

func setChainFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&testChainFlags.WriteURI, "write", "w", "http://localhost:46658/rpc", "URI for sending txs")
	fs.StringVarP(&testChainFlags.ReadURI, "read", "r", "http://localhost:46658/query", "URI for quering app state")
	fs.StringVarP(&testChainFlags.ChainID, "chain", "", "default", "chain ID")
}

func overrideChainFlags(flags chainFlags) {
	testChainFlags = flags
}

func newDeployCommand() *cobra.Command {
	var flags deployTxFlags
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, runBytecode, txReceipt, err := deployTx(flags.Bytecode, flags.PrivFile, flags.PublicFile, flags.Name)
			if err != nil {
				return err
			}
			fmt.Println("New contract deployed with address: ", addr)
			fmt.Println("Runtime bytecode: ", runBytecode)
			fmt.Println("Transcation receipt: ", txReceipt)
			return nil
		},
	}
	deployCmd.Flags().StringVarP(&flags.Bytecode, "bytecode", "b", "", "bytecode file")
	deployCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	deployCmd.Flags().StringVarP(&flags.PrivFile, "key", "k", "", "private key file")
	deployCmd.Flags().StringVarP(&flags.Name, "name", "n", "", "contract name")
	setChainFlags(deployCmd.Flags())
	return deployCmd
}

func deployTx(bcFile, privFile, pubFile, name string) (loom.Address, []byte, []byte, error) {
	clientAddr, signer, err := caller(privFile, pubFile)
	if err != nil {
		return *new(loom.Address), nil, nil, err
	}

	bytetext, err := ioutil.ReadFile(bcFile)
	if err != nil {
		return *new(loom.Address), nil, nil, err
	}
	if string(bytetext[0:2]) == "0x" {
		bytetext = bytetext[2:]
	}
	bytecode, err := hex.DecodeString(string(bytetext))
	if err != nil {
		return *new(loom.Address), nil, nil, err
	}

	rpcclient := client.NewDAppChainRPCClient(testChainFlags.ChainID, testChainFlags.WriteURI, testChainFlags.ReadURI)
	respB, err := rpcclient.CommitDeployTx(clientAddr, signer, vm.VMType_EVM, bytecode, name)
	if err != nil {
		return *new(loom.Address), nil, nil, err
	}
	response := vm.DeployResponse{}
	err = proto.Unmarshal(respB, &response)
	if err != nil {
		return *new(loom.Address), nil, nil, err
	}
	addr := loom.UnmarshalAddressPB(response.Contract)
	output := vm.DeployResponseData{}
	err = proto.Unmarshal(response.Output, &output)

	return addr, output.Bytecode, output.TxHash, err
}

type callTxFlags struct {
	ContractAddr string `json:"contractaddr"`
	ContractName string `json:"contractname"`
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
			resp, err := callTx(flags.ContractAddr, flags.ContractName, flags.Input, flags.PrivFile, flags.PublicFile)
			if err != nil {
				return err
			}
			fmt.Println("Call response: ", resp)
			return nil
		},
	}
	callCmd.Flags().StringVarP(&flags.ContractAddr, "contract-addr", "c", "", "contract address")
	callCmd.Flags().StringVarP(&flags.ContractName, "contract-name", "n", "", "contract name")
	callCmd.Flags().StringVarP(&flags.Input, "input", "i", "", "file with input data")
	callCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	callCmd.Flags().StringVarP(&flags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(callCmd.Flags())
	return callCmd
}

func callTx(addr, name, input, privFile, publicFile string) ([]byte, error) {
	rpcclient := client.NewDAppChainRPCClient(testChainFlags.ChainID, testChainFlags.WriteURI, testChainFlags.ReadURI)
	var contractAddr loom.Address
	var err error
	if addr != "" {
		if name != "" {
			fmt.Println("Both name and address entered, using address ", addr)
		}
		contractLocalAddr, err := loom.LocalAddressFromHexString(addr)
		if err != nil {
			return nil, err
		}
		contractAddr = loom.Address{
			ChainID: testChainFlags.ChainID,
			Local:   contractLocalAddr,
		}
	} else {
		contractAddr, err = rpcclient.Resolve(name)
	}
	if err != nil {
		return nil, err
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
	return rpcclient.CommitCallTx(clientAddr, contractAddr, signer, vm.VMType_EVM, incode)
}

func newStaticCallCommand() *cobra.Command {
	var flags callTxFlags

	staticCallCmd := &cobra.Command{
		Use:   "static-call",
		Short: "Calls a read-only method on an EVM contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := staticCallTx(flags.ContractAddr, flags.ContractName, flags.Input, flags.PrivFile, flags.PublicFile)
			if err != nil {
				return err
			}
			fmt.Println("Call response: ", resp)
			return nil
		},
	}
	staticCallCmd.Flags().StringVarP(&flags.ContractAddr, "contract-addr", "c", "", "contract address")
	staticCallCmd.Flags().StringVarP(&flags.ContractName, "contract-name", "n", "", "contract name")
	staticCallCmd.Flags().StringVarP(&flags.Input, "input", "i", "", "file with input data")
	staticCallCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	staticCallCmd.Flags().StringVarP(&flags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(staticCallCmd.Flags())
	return staticCallCmd
}

func staticCallTx(addr, name, input string, privFile, publicFile string) ([]byte, error) {

	rpcclient := client.NewDAppChainRPCClient(testChainFlags.ChainID, testChainFlags.WriteURI, testChainFlags.ReadURI)
	var contractLocalAddr loom.LocalAddress
	var err error
	if addr != "" {
		contractLocalAddr, err = loom.LocalAddressFromHexString(addr)
		if name != "" {
			fmt.Println("Both name and address entered, using address ", addr)
		}
	} else {
		contractAddr, err := rpcclient.Resolve(name)
		if err != nil {
			return nil, err
		}
		contractLocalAddr = contractAddr.Local
	}
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

	clientAddr, _, err := caller(privFile, publicFile)
	if err != nil {
		clientAddr = loom.Address{}
	}

	return rpcclient.QueryEvm(clientAddr, contractLocalAddr, incode)
}

func caller(privKeyB64, publicKeyB64 string) (loom.Address, auth.Signer, error) {
	privKey, err := ioutil.ReadFile(privKeyB64)
	if err != nil {
		log.Fatalf("Cannot read priv key: %s", privKeyB64)
	}
	privKey, err = base64.StdEncoding.DecodeString(string(privKey))
	if err != nil {
		log.Fatalf("Cannot decode priv file: %s", privKeyB64)
	}
	signer := auth.NewEd25519Signer(privKey)

	localAddr := []byte{}
	if len(publicKeyB64) > 0 {
		addr, err := ioutil.ReadFile(publicKeyB64)
		if err != nil {
			log.Fatalf("Cannot read address file: %s", publicKeyB64)
		}
		addr, err = base64.StdEncoding.DecodeString(string(addr))
		if err != nil {
			log.Fatalf("Cannot decode address file: %s", publicKeyB64)
		}
		localAddr = loom.LocalAddressFromPublicKey(addr)
	} else {
		localAddr = loom.LocalAddressFromPublicKey(signer.PublicKey())
	}

	clientAddr := loom.Address{
		ChainID: testChainFlags.ChainID,
		Local:   localAddr,
	}

	return clientAddr, signer, err
}
