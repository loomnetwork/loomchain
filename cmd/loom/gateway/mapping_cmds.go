// +build evm

package gateway

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const mapContractsCmdExample = `
./loom gateway map-contracts \
	0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2 0x5d1ddf5223a412d24901c32d14ef56cb706c0f64 \
	--eth-key path/to/eth_priv.key \
	--eth-tx 0x3fee8c220416862ec836e055d8261f62cd874fdfbf29b3ccba29d271c047f96c \
	--key path/to/loom_priv.key
./loom gateway map-contracts \
	0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2 0x5d1ddf5223a412d24901c32d14ef56cb706c0f64 \
	--key <base64-encoded-private-key-of-gateway-owner>
	--authorized
`

const prefixedSigLength uint64 = 66

func newMapContractsCommand() *cobra.Command {
	var txHashStr string
	var authorized bool
	var chainID string
	var gatewayType string
	cmd := &cobra.Command{
		Use:     "map-contracts <local-contract-addr> <foreign-contract-addr>",
		Short:   "Links a DAppChain token contract to an Ethereum token contract via the Transfer Gateway.",
		Example: mapContractsCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			ethKeyPath := gatewayCmdFlags.EthPrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			localContractAddr, err := hexToLoomAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "failed to resolve local contract address")
			}

			if !common.IsHexAddress(args[1]) {
				return errors.Wrap(err, "invalid foreign contract address")
			}
			foreignContractAddr := common.HexToAddress(args[1])

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(gatewayType)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			if authorized {
				req := &tgtypes.TransferGatewayAddContractMappingRequest{
					ForeignContract: loom.Address{
						ChainID: chainID,
						Local:   foreignContractAddr.Bytes(),
					}.MarshalPB(),
					LocalContract: localContractAddr.MarshalPB(),
				}

				_, err = gateway.Call("AddAuthorizedContractMapping", req, signer, nil)
				return err
			}

			creatorKey, err := crypto.LoadECDSA(ethKeyPath)
			if err != nil {
				return errors.Wrap(err, "failed to load creator Ethereum key")
			}

			hash := ssha.SoliditySHA3(
				[]string{"address", "address"},
				foreignContractAddr,
				localContractAddr.Local.String(),
			)

			var sig []byte
			if gatewayType == "tron-gateway" {
				hash = evmcompat.PrefixHeader(hash, evmcompat.SignatureType_TRON)
				sig, err = evmcompat.GenerateTypedSig(hash, creatorKey, evmcompat.SignatureType_TRON)
				if err != nil {
					return errors.Wrap(err, "failed to generate creator signature")
				}
			} else {
				sig, err = evmcompat.GenerateTypedSig(hash, creatorKey, evmcompat.SignatureType_EIP712)
				if err != nil {
					return errors.Wrap(err, "failed to generate creator signature")
				}
			}

			req := &tgtypes.TransferGatewayAddContractMappingRequest{
				ForeignContract: loom.Address{
					ChainID: chainID,
					Local:   foreignContractAddr.Bytes(),
				}.MarshalPB(),
				LocalContract:             localContractAddr.MarshalPB(),
				ForeignContractCreatorSig: sig,
			}

			if gatewayType != "tron-gateway" {
				txHash, err := hex.DecodeString(strings.TrimPrefix(txHashStr, "0x"))
				if err != nil {
					return err
				}
				req.ForeignContractTxHash = txHash
			}

			_, err = gateway.Call("AddContractMapping", req, signer, nil)
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.BoolVar(&authorized, "authorized", false, "Add contract mapping authorized by the Gateway owner")
	cmdFlags.StringVar(&txHashStr, "eth-tx", "", "Ethereum hash of contract creation tx")
	cmdFlags.StringVar(&chainID, "chain-id", "eth", "Foreign chain id")
	cmdFlags.StringVar(&gatewayType, "gateway", "gateway", "Gateway name: gateway, loomcoin-gateway, or tron-gateway")
	return cmd
}

const mapBinanceContractsCmdExample = `
./loom gateway map-binance-contracts \
	0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2 LOOM-172 \
	--eth-key path/to/binance_priv.key \
	--key path/to/loom_priv.key
./loom gateway map-binance-contracts \
	0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2 LOOM-172 \
	--key <base64-encoded-private-key-of-gateway-owner>
	--authorized
`

func newMapBinanceContractsCommand() *cobra.Command {
	var authorized bool
	var chainID string
	var gatewayType = "binance-gateway"
	cmd := &cobra.Command{
		Use:     "map-binance-contracts <local-contract-addr> <token-name>",
		Short:   "Links a DAppChain token contract to an Binance token via the Transfer Gateway.",
		Example: mapBinanceContractsCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			ethKeyPath := gatewayCmdFlags.EthPrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			localContractAddr, err := hexToLoomAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "failed to resolve local contract address")
			}

			tokenNameHex := hex.EncodeToString([]byte(args[1]))
			foreignContractAddr := common.HexToAddress(tokenNameHex)

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(gatewayType)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			if authorized {
				req := &tgtypes.TransferGatewayAddContractMappingRequest{
					ForeignContract: loom.Address{
						ChainID: chainID,
						Local:   foreignContractAddr.Bytes(),
					}.MarshalPB(),
					LocalContract: localContractAddr.MarshalPB(),
				}

				_, err = gateway.Call("AddAuthorizedContractMapping", req, signer, nil)
				return err
			}

			creatorKey, err := crypto.LoadECDSA(ethKeyPath)
			if err != nil {
				return errors.Wrap(err, "failed to load creator Ethereum key")
			}

			hash := ssha.SoliditySHA3(
				[]string{"address", "address"},
				foreignContractAddr,
				localContractAddr.Local.String(),
			)
			sig, err := evmcompat.GenerateTypedSig(hash, creatorKey, evmcompat.SignatureType_BINANCE)
			if err != nil {
				return errors.Wrap(err, "failed to generate creator signature")
			}

			// no ForeignContractTxHash for binance
			req := &tgtypes.TransferGatewayAddContractMappingRequest{
				ForeignContract: loom.Address{
					ChainID: chainID,
					Local:   foreignContractAddr.Bytes(),
				}.MarshalPB(),
				LocalContract:             localContractAddr.MarshalPB(),
				ForeignContractCreatorSig: sig,
			}

			_, err = gateway.Call("AddContractMapping", req, signer, nil)
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.BoolVar(&authorized, "authorized", false, "Add contract mapping authorized by the Gateway owner")
	cmdFlags.StringVar(&chainID, "chain-id", "binance", "Foreign chain id")
	return cmd
}

const mapAccountsCmdExample = `
./loom gateway map-accounts	--key path/to/loom_priv.key --eth-key path/to/eth_priv.key OR
./loom gateway map-accounts --interactive --key path/to/loom_priv.key --eth-address <your-eth-address>
`

const mapAccountsConfirmationMsg = `
Mapping Accounts
%v <-> %v
Are you sure? [y/n]
`

func newMapAccountsCommand() *cobra.Command {
	var ethAddressStr string
	var silent, interactive bool
	cmd := &cobra.Command{
		Use:     "map-accounts",
		Short:   "Links a DAppChain account to an Ethereum account via the Transfer Gateway.",
		Example: mapAccountsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			ethKeyPath := gatewayCmdFlags.EthPrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			localOwnerAddr := loom.Address{
				ChainID: gatewayCmdFlags.ChainID,
				Local:   loom.LocalAddressFromPublicKey(signer.PublicKey()),
			}

			rpcClient := getDAppChainClient()
			mapperAddr, err := rpcClient.Resolve("addressmapper")
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Address Mapper address")
			}
			mapper := client.NewContract(rpcClient, mapperAddr.Local)
			mappedAccount, err := getMappedAccount(mapper, localOwnerAddr)
			if err == nil {
				return fmt.Errorf("Account %v is already mapped to %v", localOwnerAddr, mappedAccount)
			}

			var foreignOwnerAddr loom.Address
			var req *amtypes.AddressMapperAddIdentityMappingRequest
			if !interactive {
				// get it from the key
				ethOwnerKey, err := crypto.LoadECDSA(ethKeyPath)
				if err != nil {
					return errors.Wrap(err, "failed to load owner Ethereum key")
				}

				foreignOwnerAddr = loom.Address{
					ChainID: "eth",
					Local:   crypto.PubkeyToAddress(ethOwnerKey.PublicKey).Bytes(),
				}

				hash := ssha.SoliditySHA3(
					[]string{"address", "address"},
					localOwnerAddr.Local.String(),
					foreignOwnerAddr.Local.String(),
				)

				sign, err := evmcompat.GenerateTypedSig(hash, ethOwnerKey, evmcompat.SignatureType_EIP712)
				if err != nil {
					return errors.Wrap(err, "failed to generate foreign owner signature")
				}

				req = &amtypes.AddressMapperAddIdentityMappingRequest{
					From:      localOwnerAddr.MarshalPB(),
					To:        foreignOwnerAddr.MarshalPB(),
					Signature: sign,
				}

				ethAddressStr = crypto.PubkeyToAddress(ethOwnerKey.PublicKey).String()
			} else {
				addr, err := loom.LocalAddressFromHexString(ethAddressStr)
				if err != nil {
					return errors.Wrap(err, "invalid ethAddressStr")
				}
				foreignOwnerAddr = loom.Address{
					ChainID: "eth",
					Local:   addr,
				}

				hash := ssha.SoliditySHA3(
					[]string{"address", "address"},
					localOwnerAddr.Local.String(),
					foreignOwnerAddr.Local.String(),
				)

				sign, err := getSignatureInteractive(hash, evmcompat.SignatureType_GETH)
				if err != nil {
					return err
				}
				// allow only SignatureType_GETH for the recover address function
				allowSigTypes := []evmcompat.SignatureType{evmcompat.SignatureType_GETH}
				// Do a local recovery on the signature to make sure the user is passing the correct byte
				signer, err := evmcompat.RecoverAddressFromTypedSig(hash, sign[:], allowSigTypes)
				if err != nil {
					return err
				}
				fmt.Println("GOT SIGNER", signer.String())

				req = &amtypes.AddressMapperAddIdentityMappingRequest{
					From:      localOwnerAddr.MarshalPB(),
					To:        foreignOwnerAddr.MarshalPB(),
					Signature: sign[:],
				}
			}

			if !silent {
				fmt.Printf(mapAccountsConfirmationMsg, localOwnerAddr, ethAddressStr)
				var input string
				n, err := fmt.Scan(&input)
				if err != nil {
					return err
				}
				if n != 1 {
					return errors.New("expected y/n")
				}
				if strings.ToLower(input) != "y" && strings.ToLower(input) != "yes" {
					return nil
				}
			}

			_, err = mapper.Call("AddIdentityMapping", req, signer, nil)

			if err == nil {
				fmt.Println("...Address has been successfully mapped!")
			}
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&ethAddressStr, "eth-address", "", "Ethereum address of account owner")
	cmdFlags.BoolVar(&silent, "silent", false, "Don't ask for address confirmation")
	cmdFlags.BoolVar(&interactive, "interactive", false, "Make the mapping of an account interactive by requiring the signature to be provided by the user instead of signing inside the client.")
	return cmd
}

const ListContractMappingCmdExample = `
loom gateway list-contract-mappings
loom gateway list-contract-mappings --raw
loom gateway list-contract-mappings --json
`

func newListContractMappingsCommand() *cobra.Command {
	var gatewayType string
	var formatJson bool
	var formatRaw bool
	cmd := &cobra.Command{
		Use:     "list-contract-mappings",
		Short:   "List all contract mappings",
		Example: ListContractMappingCmdExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(gatewayType)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)
			req := &tgtypes.TransferGatewayListContractMappingRequest{}
			resp := &tgtypes.TransferGatewayListContractMappingResponse{}
			_, err = gateway.StaticCall("ListContractMapping", req, gatewayAddr, resp)
			if err != nil {
				return errors.Wrap(err, "failed to call gateway.ListContractMapping")
			}

			if formatJson {
				type formattedMapping struct {
					Local   string `json:"local"`
					Foreign string `json:"foreign"`
				}

				mappingData := struct {
					Confirmed []formattedMapping `json:"confirmed"`
					Pending   []formattedMapping `json:"pending"`
				}{
					Confirmed: make([]formattedMapping, 0, len(resp.ConfimedMappings)),
					Pending:   make([]formattedMapping, 0, len(resp.PendingMappings)),
				}
				for _, value := range resp.ConfimedMappings {
					mappingData.Confirmed = append(mappingData.Confirmed, formattedMapping{
						Local:   loom.UnmarshalAddressPB(value.From).Local.String(),
						Foreign: loom.UnmarshalAddressPB(value.To).Local.String(),
					})
				}
				for _, value := range resp.PendingMappings {
					mappingData.Pending = append(mappingData.Pending, formattedMapping{
						Local:   loom.UnmarshalAddressPB(value.LocalContract).Local.String(),
						Foreign: loom.UnmarshalAddressPB(value.ForeignContract).Local.String(),
					})
				}
				bytes, err := json.MarshalIndent(mappingData, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(bytes))
				return nil
			} else if formatRaw {
				out, err := formatJSON(resp)
				if err != nil {
					return err
				}
				fmt.Println(out)
			} else {
				ml := struct {
					From   int
					To     int
					Status int
				}{
					From:   50,
					To:     50,
					Status: 9,
				}
				for _, value := range resp.PendingMappings {
					if len(loom.UnmarshalAddressPB(value.ForeignContract).String()) > ml.From {
						ml.From = len(loom.UnmarshalAddressPB(value.ForeignContract).String())
					}
					if len(loom.UnmarshalAddressPB(value.LocalContract).String()) > ml.To {
						ml.To = len(loom.UnmarshalAddressPB(value.LocalContract).String())
					}
				}
				for _, value := range resp.ConfimedMappings {
					if len(loom.UnmarshalAddressPB(value.From).String()) > ml.From {
						ml.From = len(loom.UnmarshalAddressPB(value.From).String())
					}
					if len(loom.UnmarshalAddressPB(value.To).String()) > ml.To {
						ml.To = len(loom.UnmarshalAddressPB(value.To).String())
					}
				}

				fmt.Printf("%-*s | %-*s | %-*s\n", ml.From, "From", ml.To, "To", ml.Status, "Status")
				for _, value := range resp.PendingMappings {
					fmt.Printf("%-*s | %-*s | %-*s\n", ml.From, loom.UnmarshalAddressPB(value.ForeignContract).String(), ml.To, loom.UnmarshalAddressPB(value.LocalContract).String(), ml.Status, "PENDING")
				}
				for _, value := range resp.ConfimedMappings {
					fmt.Printf("%-*s | %-*s | %-*s\n", ml.From, loom.UnmarshalAddressPB(value.From).String(), ml.To, loom.UnmarshalAddressPB(value.To).String(), ml.Status, "CONFIRMED")
				}
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&gatewayType, "gateway", "gateway", "Gateway name: gateway, loomcoin-gateway, or tron-gateway")
	cmdFlags.BoolVar(&formatRaw, "raw", false, "Output raw JSON")
	cmdFlags.BoolVar(&formatJson, "json", false, "Output prettified JSON")
	return cmd
}

const getContractMappingCmdExample = `
loom gateway get-contract-mapping 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

type Mapping struct {
	Address   string `json:"address"`
	IsPending bool   `json:"is_pending"`
	Found     bool   `json:"found"`
}

func newGetContractMappingCommand() *cobra.Command {
	var gatewayType string
	cmd := &cobra.Command{
		Use:     "get-contract-mapping <contract-addr>",
		Short:   "Get Contract Mapping",
		Example: getContractMappingCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var mapping Mapping
			var contractAddr loom.Address
			var err error
			contractAddr, err = cli.ParseAddress(args[0], gatewayCmdFlags.ChainID)
			if err != nil {
				return err
			}
			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(gatewayType)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)
			req := &tgtypes.TransferGatewayGetContractMappingRequest{
				From: contractAddr.MarshalPB(),
			}
			resp := &tgtypes.TransferGatewayGetContractMappingResponse{}
			_, err = gateway.StaticCall("GetContractMapping", req, gatewayAddr, resp)
			if err != nil {
				return errors.Wrap(err, "failed to call gateway.GetContractMapping")
			}
			if resp.MappedAddress != nil {
				mapping.Address = loom.UnmarshalAddressPB(resp.MappedAddress).String()
				mapping.IsPending = resp.IsPending
				mapping.Found = resp.Found
			} else {
				fmt.Println("No mapping found")
				return nil
			}
			type maxLength struct {
				From   int
				To     int
				Status int
			}
			ml := maxLength{From: 50, To: 50, Status: 9}
			if len(contractAddr.String()) > ml.From {
				ml.From = len(contractAddr.String())
			}
			if len(mapping.Address) > ml.To {
				ml.To = len(mapping.Address)
			}
			fmt.Printf("%-*s | %-*s | %-*s\n", ml.From, "From", ml.To, "To", ml.Status, "Status")
			if mapping.IsPending {
				fmt.Printf("%-*s | %-*s | %-*s\n", ml.From, contractAddr, ml.To, mapping.Address, ml.Status, "PENDING")
			} else {
				fmt.Printf("%-*s | %-*s | %-*s\n", ml.From, contractAddr, ml.To, mapping.Address, ml.Status, "CONFIRMED")
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&gatewayType, "gateway", "gateway", "Gateway name: gateway, loomcoin-gateway, or tron-gateway")
	return cmd
}

func getSignatureInteractive(hash []byte, sigType evmcompat.SignatureType) ([prefixedSigLength]byte, error) {
	// get it from the signature
	fmt.Printf("Please paste the following hash to your signing software. After signing it, paste the signature below (prefixed with 0x)\n")
	fmt.Printf("0x%v\n", hex.EncodeToString(hash))

	var sig string
	fmt.Print("> ")
	n, err := fmt.Scan(&sig)
	if err != nil {
		return [66]byte{}, err
	}
	if n != 1 {
		return [66]byte{}, errors.New("invalid signature")
	}

	// todo: check if prefixed with 0x
	sigStripped, err := hex.DecodeString(sig[2:])
	if err != nil {
		return [66]byte{}, errors.New("please paste the signature prefixed with 0x")
	}

	// increase by 27 in case recovery id was invalid
	if sigStripped[64] == 0 || sigStripped[64] == 1 {
		sigStripped[64] += 27
	}

	// create the prefixed sig so that it matches the way it's verified on address mapper
	var sigBytes [prefixedSigLength]byte
	prefix := byte(sigType)
	typedSig := append(make([]byte, 0, prefixedSigLength), prefix)
	copy(sigBytes[:], append(typedSig, sigStripped...))

	return sigBytes, nil
}

func getMappedAccount(mapper *client.Contract, account loom.Address) (loom.Address, error) {
	req := &amtypes.AddressMapperGetMappingRequest{
		From: account.MarshalPB(),
	}
	resp := &amtypes.AddressMapperGetMappingResponse{}
	_, err := mapper.StaticCall("GetMapping", req, account, resp)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(resp.To), nil
}

func getDAppChainClient() *client.DAppChainRPCClient {
	writeURI := gatewayCmdFlags.URI + "/rpc"
	readURI := gatewayCmdFlags.URI + "/query"
	return client.NewDAppChainRPCClient(gatewayCmdFlags.ChainID, writeURI, readURI)
}

const ListPendingDepositedUserHotWallet = `
loom gateway list-deposited-user-hot-wallet
`

func newGetPendingDepositedUserHotWalletCommand() *cobra.Command {
	var gatewayType string
	cmd := &cobra.Command{
		Use:     "list-deposited-user-hot-wallet",
		Short:   "List all user hot-wallets that has balance",
		Example: ListPendingDepositedUserHotWallet,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(gatewayType)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)
			req := &tgtypes.TransferGatewayGetUserHotWalletRequest{}
			resp := &tgtypes.TransferGatewayListPendingUserHotWalletDepositedResponse{}

			_, err = gateway.StaticCall("ListPendingUserHotWalletDeposited", req, gatewayAddr, resp)
			if err != nil {
				return errors.Wrap(err, "failed to call gateway.ListPendingUserHotWalletDeposited")
			}
			out, err := formatJSON(resp)
			if err != nil {
				return err
			}

			if err := ioutil.WriteFile("deposited_user_hot_wallet.json", []byte(out), 0664); err != nil {
				return fmt.Errorf("Unable to write output file: %v", err)
			}

			fmt.Println("Exported to deposited_user_hot_wallet.json")
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&gatewayType, "gateway", "gateway", "Gateway name: gateway, loomcoin-gateway, or tron-gateway")
	return cmd
}
