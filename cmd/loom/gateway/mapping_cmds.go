// +build evm

package gateway

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"strings"

	"github.com/loomnetwork/go-loom"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom/auth"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const mapContractsCmdExample = `
./loom gateway map-contracts \
	0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2 0x5d1ddf5223a412d24901c32d14ef56cb706c0f64 \
	--eth-account 0x90A3D2aFf8C3c60614D40B034FEc77B465AD92D5 \
	--eth-key file://path/to/eth_priv.key \
	--eth-tx 0x3fee8c220416862ec836e055d8261f62cd874fdfbf29b3ccba29d271c047f96c \
	--key file://path/to/loom_priv.key
`

func newMapContractsCommand() *cobra.Command {
	var creatorAddrStr, loomKeyStr, ethKeyStr, txHashStr string
	cmd := &cobra.Command{
		Use:     "map-contracts <local-contract-addr> <foreign-contract-addr>",
		Short:   "Links a DAppChain token contract to an Ethereum token contract via the Transfer Gateway.",
		Example: mapContractsCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			creatorKey, err := getEthereumPrivateKey(ethKeyStr)
			if err != nil {
				return errors.Wrap(err, "failed to load creator Ethereum key")
			}

			signer, err := getDAppChainSigner(loomKeyStr)
			if err != nil {
				return errors.Wrap(err, "failed to load creator DAppChain key")
			}

			rpcClient := getDAppChainClient()
			localContractAddr, err := hexToLoomAddress(rpcClient, args[0])
			if err != nil {
				return errors.Wrap(err, "failed to resolve local contract address")
			}
			if !common.IsHexAddress(args[1]) {
				return errors.Wrap(err, "invalid foreign contract address")
			}
			foreignContractAddr := common.HexToAddress(args[1])

			gatewayAddr, err := rpcClient.Resolve("gateway")
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			txHash, err := hex.DecodeString(strings.TrimPrefix(txHashStr, "0x"))
			if err != nil {
				return err
			}

			hash := ssha.SoliditySHA3(
				ssha.Address(foreignContractAddr),
				ssha.Address(common.BytesToAddress(localContractAddr.Local)),
			)
			sig, err := evmcompat.GenerateTypedSig(hash, creatorKey, evmcompat.SignatureType_EIP712)
			if err != nil {
				return errors.Wrap(err, "failed to generate creator signature")
			}

			req := &tgtypes.TransferGatewayAddContractMappingRequest{
				ForeignContract: loom.Address{
					ChainID: "eth",
					Local:   foreignContractAddr.Bytes(),
				}.MarshalPB(),
				LocalContract:             localContractAddr.MarshalPB(),
				ForeignContractCreatorSig: sig,
				ForeignContractTxHash:     txHash,
			}

			_, err = gateway.Call("AddContractMapping", req, signer, nil)
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&creatorAddrStr, "eth-account", "", "Ethereum address of contract creator")
	cmdFlags.StringVar(&ethKeyStr, "eth-key", "", "Ethereum private key of contract creator")
	cmdFlags.StringVar(&txHashStr, "eth-tx", "", "Ethereum hash of contract creation tx")
	cmdFlags.StringVarP(&loomKeyStr, "key", "k", "", "DAppChain private key of contract creator")
	cmd.MarkFlagRequired("eth-account")
	cmd.MarkFlagRequired("eth-key")
	cmd.MarkFlagRequired("eth-tx")
	cmd.MarkFlagRequired("key")
	return cmd
}

// Loads the given Ethereum private key.
// privateKeyPath can either be a hex-encoded string representing the key, or the path to a file
// containing the hex-encoded key, in the latter case the path must be prefixed by file://
// (e.g. file://path/to/some.key)
func getEthereumPrivateKey(privateKeyPath string) (*ecdsa.PrivateKey, error) {
	keyStr := privateKeyPath
	if strings.HasPrefix(privateKeyPath, "file://") {
		hexStr, err := ioutil.ReadFile(strings.TrimPrefix(privateKeyPath, "file://"))
		if err != nil {
			return nil, errors.Wrap(err, "failed to load key file")
		}
		keyStr = string(hexStr)
	}

	return crypto.HexToECDSA(strings.TrimPrefix(keyStr, "0x"))
}

// Loads the given DAppChain private key.
// privateKeyPath can either be a base64-encoded string representing the key, or the path to a file
// containing the base64-encoded key, in the latter case the path must be prefixed by file://
// (e.g. file://path/to/some.key)
func getDAppChainSigner(privateKeyPath string) (auth.Signer, error) {
	keyStr := privateKeyPath
	if strings.HasPrefix(privateKeyPath, "file://") {
		b64, err := ioutil.ReadFile(strings.TrimPrefix(privateKeyPath, "file://"))
		if err != nil {
			return nil, errors.Wrap(err, "failed to load key file")
		}
		keyStr = string(b64)
	}

	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 key file")
	}
	signer := auth.NewEd25519Signer(keyBytes)
	return signer, nil
}

func getDAppChainClient() *client.DAppChainRPCClient {
	writeURI := gatewayCmdFlags.URI + "/rpc"
	readURI := gatewayCmdFlags.URI + "/query"
	return client.NewDAppChainRPCClient(gatewayCmdFlags.ChainID, writeURI, readURI)
}
