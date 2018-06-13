package gateway

import (
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	ltypes "github.com/loomnetwork/go-loom/types"
	gc "github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type OracleConfig struct {
	// URI of an Ethereum node
	EthereumURI string
	// Gateway contract address on Ethereum
	GatewayHexAddress string
	ChainID           string
	WriteURI          string
	ReadURI           string
}

type Oracle struct {
	cfg        OracleConfig
	solGateway *Gateway
	goGateway  *client.Contract
	startBlock uint64
}

func NewOracle(cfg OracleConfig) *Oracle {
	return &Oracle{cfg: cfg}
}

func (orc *Oracle) Init() error {
	cfg := &orc.cfg
	con, err := ethclient.Dial(cfg.EthereumURI)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Ethereum")
	}

	orc.solGateway, err = NewGateway(common.HexToAddress(cfg.GatewayHexAddress), con)
	if err != nil {
		return errors.Wrap(err, "failed to bind Gateway Solidity contract")
	}

	dappClient := client.NewDAppChainRPCClient(cfg.ChainID, cfg.WriteURI, cfg.ReadURI)
	contractAddr, err := dappClient.Resolve("gateway")
	if err != nil {
		return errors.Wrap(err, "failed to resolve Gateway Go contract address")
	}
	orc.goGateway = client.NewContract(dappClient, contractAddr.Local)
	return nil
}

func (orc *Oracle) Run() {
	// TODO: the validator's key should be passed in through the config
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatal(err)
	}
	signer := auth.NewEd25519Signer([]byte(privKey))
	// TODO: query orc.goGateway for the last block and use that +1 as the start block
	orc.startBlock = 0
	for {
		batch, err := orc.fetchEvents()
		// TODO: probably shouldn't be fatal
		if err != nil {
			log.Fatal(err)
		}
		if _, err := orc.goGateway.Call("ProcessEventBatch", batch, signer, nil); err != nil {
			log.Fatal(err)
		}
		// TODO: should be configurable
		time.Sleep(5 * time.Second)
	}
}

func (orc *Oracle) fetchEvents() (*gc.ProcessEventBatchRequest, error) {
	// NOTE: Currently either all blocks from w.StartBlock are processed successfully or none are.
	lastBlock := orc.startBlock - 1

	ftDeposits := []*gc.FTDeposit{}
	nftDeposits := []*gc.NFTDeposit{}

	// TODO: Currently there are 3 separate requests being made, should just make one for all 3 events
	//       because it would be (a) more efficient, and (b) simplify the code a fair bit since
	//       you wouldn't have to track which block range has been processed for each event type.
	ethIt, err := orc.solGateway.FilterETHReceived(&bind.FilterOpts{Start: orc.startBlock})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ETHReceived")
	}
	for {
		ok := ethIt.Next()
		if ok {
			ev := ethIt.Event
			fmt.Printf("ETHReceived: %v from %v in block %v\n",
				ev.Amount.String(), ev.From.Hex(), ev.Raw.BlockNumber)
			tokenAddr := loom.RootAddress("eth")
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ETHReceived from address")
			}
			ftDeposits = append(ftDeposits, &gc.FTDeposit{
				Token:  tokenAddr.MarshalPB(),
				From:   loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
				Amount: &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
			})
			if lastBlock < ev.Raw.BlockNumber {
				lastBlock = ev.Raw.BlockNumber
			}
		} else {
			err := ethIt.Error()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get event data for ETHReceived")
			}
			ethIt.Close()
			break
		}
	}

	erc20It, err := orc.solGateway.FilterERC20Received(&bind.FilterOpts{Start: orc.startBlock})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ERC20Received")
	}
	for {
		ok := erc20It.Next()
		if ok {
			ev := erc20It.Event
			fmt.Printf("ERC20Received: %v from %v in block %v\n",
				ev.Amount.String(), ev.From.Hex(), ev.Raw.BlockNumber)
			// TODO: fill in the actual token address
			tokenAddr := loom.RootAddress("blah")
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC20Received from address")
			}
			ftDeposits = append(ftDeposits, &gc.FTDeposit{
				Token:  tokenAddr.MarshalPB(),
				From:   loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
				Amount: &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
			})
			if lastBlock < ev.Raw.BlockNumber {
				lastBlock = ev.Raw.BlockNumber
			}
		} else {
			err := erc20It.Error()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get event data for ERC20Received")
			}
			erc20It.Close()
			break
		}
	}

	erc721It, err := orc.solGateway.FilterERC721Received(&bind.FilterOpts{Start: orc.startBlock})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ERC721Received")
	}
	for {
		ok := erc721It.Next()
		if ok {
			ev := erc721It.Event
			fmt.Printf("ERC721Received: %v from %v in block %v\n",
				ev.Uid.String(), ev.From.Hex(), ev.Raw.BlockNumber)
			localAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721Received from address")
			}
			nftDeposits = append(nftDeposits, &gc.NFTDeposit{
				Token: loom.RootAddress("eth").MarshalPB(),
				From:  loom.Address{ChainID: "eth", Local: localAddr}.MarshalPB(),
				Uid:   &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Uid)},
			})
			if lastBlock < ev.Raw.BlockNumber {
				lastBlock = ev.Raw.BlockNumber
			}
		} else {
			err := erc721It.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for ERC721Received")
			}
			erc721It.Close()
			break
		}
	}

	orc.startBlock = lastBlock + 1
	return &gc.ProcessEventBatchRequest{
		FtDeposits:  ftDeposits,
		NftDeposits: nftDeposits,
	}, nil
}
