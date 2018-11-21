package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/vm"
)

type deployTxFlags struct {
	Bytecode   string `json:"bytecode"`
	PublicFile string `json:"publicfile"`
	PrivFile   string `json:"privfile"`
	Name       string `json:"name"`
}

func setChainFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&cli.TxFlags.WriteURI, "write", "w", "http://localhost:46658/rpc", "URI for sending txs")
	fs.StringVarP(&cli.TxFlags.ReadURI, "read", "r", "http://localhost:46658/query", "URI for quering app state")
	fs.StringVarP(&cli.TxFlags.ChainID, "chain", "", "default", "chain ID")
}

func newDeployGoCommand() *cobra.Command {
	var code string
	var flags deployTxFlags
	deployCmd := &cobra.Command{
		Use:   "deploy-go",
		Short: "Deploy a go contract from json file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deployGoTx(code, flags.PrivFile, flags.PublicFile)
		},
	}
	deployCmd.Flags().StringVarP(&code, "json-init-code", "b", "", "deploy go contract from json init file")
	deployCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	deployCmd.Flags().StringVarP(&flags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(deployCmd.Flags())
	return deployCmd
}

func deployGoTx(initFile, privFile, pubFile string) error {
	clientAddr, signer, err := caller(privFile, pubFile)
	if err != nil {
		return errors.Wrapf(err, "initialization failed")
	}
	if signer == nil {
		return fmt.Errorf("invalid private key")
	}

	gen, err := readGenesis(initFile)
	if len(gen.Contracts) == 0 {
		return fmt.Errorf("no contracts in file %s", initFile)
	}
	fmt.Printf("Attempting to deploy %v contracts\n", len(gen.Contracts))
	rpcclient := client.NewDAppChainRPCClient(cli.TxFlags.ChainID, cli.TxFlags.WriteURI, cli.TxFlags.ReadURI)

	numDeployed := 0
	for _, contract := range gen.Contracts {
		fmt.Println("Attempting to deploy contract ", contract.Name)
		_, err := rpcclient.Resolve(contract.Name)
		if err == nil {
			fmt.Printf("Contract %s already registered. Skipping\n", contract.Name)
			continue
		} else if !strings.Contains(err.Error(), registry.ErrNotFound.Error()) {
			fmt.Printf("Could not confirm contract %s regestration status, error %v. Skipping\n", contract.Name, err)
			continue
		}

		loader := codeLoaders[contract.Format]
		initCode, err := loader.LoadContractCode(
			contract.Location,
			contract.Init,
		)

		respB, err := rpcclient.CommitDeployTx(clientAddr, signer, vm.VMType_PLUGIN, initCode, contract.Name)

		if err != nil {
			fmt.Printf("Error, %v, deploying contact %s\n", err, contract.Name)
			continue
		}
		fmt.Println("Successfully deployed contract", contract.Name)
		numDeployed++
		response := vm.DeployResponse{}
		err = proto.Unmarshal(respB, &response)
		if err != nil {
			fmt.Printf("Error, %v, unmarshalling contact response %v\n", err, response)
			continue
		}
		addr := loom.UnmarshalAddressPB(response.Contract)
		fmt.Printf("Contract %s deplyed to address %s\n", contract.Name, addr.String())
	}
	fmt.Printf("%v contract(s) succesfully deployed\n", numDeployed)
	return nil
}

func newDeployCommand() *cobra.Command {
	var flags deployTxFlags
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, runBytecode, txReceipt, err := deployTx(flags.Bytecode, cli.TxFlags.PrivFile, flags.PublicFile, flags.Name)
			if err != nil {
				return err
			}
			fmt.Println("New contract deployed with address: ", addr)
			fmt.Println("Runtime bytecode: ", runBytecode)
			fmt.Println("Transaction receipt: ", txReceipt)
			return nil
		},
	}
	deployCmd.Flags().StringVarP(&flags.Bytecode, "bytecode", "b", "", "bytecode file")
	deployCmd.Flags().StringVarP(&flags.PublicFile, "address", "a", "", "address file")
	deployCmd.Flags().StringVarP(&flags.Name, "name", "n", "", "contract name")
	deployCmd.Flags().StringVarP(&cli.TxFlags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(deployCmd.Flags())
	return deployCmd
}

func deployTx(bcFile, privFile, pubFile, name string) (loom.Address, []byte, []byte, error) {
	clientAddr, signer, err := caller(privFile, pubFile)
	if err != nil {
		return *new(loom.Address), nil, nil, errors.Wrapf(err, "initialization failed")
	}
	if signer == nil {
		return *new(loom.Address), nil, nil, fmt.Errorf("invalid private key")
	}

	bytetext, err := ioutil.ReadFile(bcFile)
	if err != nil {
		return *new(loom.Address), nil, nil, errors.Wrapf(err, "reading deployment file")
	}
	if string(bytetext[0:2]) == "0x" {
		bytetext = bytetext[2:]
	}
	bytecode, err := hex.DecodeString(string(bytetext))
	if err != nil {
		return *new(loom.Address), nil, nil, errors.Wrapf(err, "decoding the data in deployment file")
	}

	rpcclient := client.NewDAppChainRPCClient(cli.TxFlags.ChainID, cli.TxFlags.WriteURI, cli.TxFlags.ReadURI)
	respB, err := rpcclient.CommitDeployTx(clientAddr, signer, vm.VMType_EVM, bytecode, name)
	if err != nil {
		return *new(loom.Address), nil, nil, errors.Wrapf(err, "CommitDeployTx")
	}
	response := vm.DeployResponse{}
	err = proto.Unmarshal(respB, &response)
	if err != nil {
		return *new(loom.Address), nil, nil, errors.Wrapf(err, "unmarshalling response")
	}
	addr := loom.UnmarshalAddressPB(response.Contract)
	output := vm.DeployResponseData{}
	err = proto.Unmarshal(response.Output, &output)

	return addr, output.Bytecode, output.TxHash, errors.Wrapf(err, "unmarshalling output")
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
			resp, err := callTx(flags.ContractAddr, flags.ContractName, flags.Input, cli.TxFlags.PrivFile, flags.PublicFile)
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
	callCmd.PersistentFlags().StringVarP(&cli.TxFlags.PrivFile, "key", "k", "", "private key file")
	setChainFlags(callCmd.PersistentFlags())
	return callCmd
}

func callTx(addr, name, input, privFile, publicFile string) ([]byte, error) {
	rpcclient := client.NewDAppChainRPCClient(cli.TxFlags.ChainID, cli.TxFlags.WriteURI, cli.TxFlags.ReadURI)
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
			ChainID: cli.TxFlags.ChainID,
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
	if signer == nil {
		return nil, fmt.Errorf("invalid private key")
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

	rpcclient := client.NewDAppChainRPCClient(cli.TxFlags.ChainID, cli.TxFlags.WriteURI, cli.TxFlags.ReadURI)
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

	clientAddr, _, _ := caller(privFile, publicFile)

	return rpcclient.QueryEvm(clientAddr, contractLocalAddr, incode)
}

type getBlockByNumerTxFlags struct {
	Full   bool   `json:"full"`
	Number string `json:"number"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

func newGetBlocksByNumber() *cobra.Command {
	var flags getBlockByNumerTxFlags

	getBlocksByNumberCmd := &cobra.Command{
		Use:   "getblocksbynumber",
		Short: "gets info for block or range of blocks number",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getBlockByNumber(flags.Number, flags.Start, flags.End, flags.Full)
		},
	}
	getBlocksByNumberCmd.Flags().BoolVarP(&flags.Full, "full", "f", false, "show full block information")
	getBlocksByNumberCmd.Flags().StringVarP(&flags.Number, "number", "n", "", "block nmber, integer, latest or pending")
	getBlocksByNumberCmd.Flags().StringVarP(&flags.Start, "start", "s", "", "start of range of blocks to return")
	getBlocksByNumberCmd.Flags().StringVarP(&flags.End, "end", "e", "", "end of range of blocks to return")
	setChainFlags(getBlocksByNumberCmd.Flags())
	return getBlocksByNumberCmd
}

func getBlockByNumber(number, start, end string, full bool) error {
	rpcclient := client.NewDAppChainRPCClient(cli.TxFlags.ChainID, cli.TxFlags.WriteURI, cli.TxFlags.ReadURI)
	if len(number) > 0 {
		resp, err := rpcclient.GetEvmBlockByNumber(number, full)
		if err != nil {
			errors.Wrap(err, "calling GetEvmBlockByNumber")
		}
		blockInfo, err := formatJSON(&resp)
		if err != nil {
			return errors.Wrap(err, "formatting block info")
		}
		fmt.Printf("Print information for block %s ", number)
		fmt.Println(blockInfo)
	} else {
		startN, err := strconv.ParseUint(start, 10, 64)
		if err != nil {
			return errors.Wrap(err, "paring start value")
		}
		endN, err := strconv.ParseUint(end, 10, 64)
		if err != nil {
			return errors.Wrap(err, "paring end value")
		}
		fmt.Printf("Print inormation for blocks %s to %s ", start, end)
		for block := startN; block <= endN; block++ {
			blockS := strconv.FormatUint(block, 10)
			resp, err := rpcclient.GetEvmBlockByNumber(blockS, full)
			if err != nil {
				return errors.Wrap(err, "calling GetEvmBlockByNumber")
			}
			blockInfo, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "formatting block info")
			}
			fmt.Printf("Print information for block %s ", blockS)
			fmt.Println(blockInfo)
			//time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

func caller(privKeyB64, publicKeyB64 string) (loom.Address, auth.Signer, error) {
	localAddr := []byte{}
	if len(publicKeyB64) > 0 {
		addr, err := ioutil.ReadFile(publicKeyB64)
		if err == nil {
			addr, err = base64.StdEncoding.DecodeString(string(addr))
			if err != nil {
				addr = []byte{}
			} else {
				localAddr = loom.LocalAddressFromPublicKey(addr)
			}
		}
	}
	var signer auth.Signer
	signer = nil
	if len(privKeyB64) > 0 {
		privKey, err := ioutil.ReadFile(privKeyB64)
		if err != nil {
			return loom.RootAddress("default"), nil, fmt.Errorf("Cannot read priv key: %s", privKeyB64)
		}
		privKey, err = base64.StdEncoding.DecodeString(string(privKey))
		if err != nil {
			return loom.RootAddress("default"), nil, fmt.Errorf("Cannot decode priv file: %s", privKeyB64)
		}

		signer := auth.NewSigner(privKey)
		if len(localAddr) == 0 {
			localAddr = loom.LocalAddressFromPublicKey(signer.PublicKey())
		}
	}
	var clientAddr loom.Address
	if len(localAddr) == 0 {
		clientAddr = loom.RootAddress(cli.TxFlags.ChainID)
	} else {
		clientAddr = loom.Address{
			ChainID: cli.TxFlags.ChainID,
			Local:   localAddr,
		}
	}

	return clientAddr, signer, nil
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
