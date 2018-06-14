package gateway

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	ltypes "github.com/loomnetwork/go-loom/types"
	gwc "github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	log "github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
)

type OracleConfig struct {
	// URI of an Ethereum node
	EthereumURI string
	// Gateway contract address on Ethereum
	GatewayHexAddress string
	ChainID           string
	WriteURI          string
	ReadURI           string
	Signer            auth.Signer
}

type Oracle struct {
	cfg        OracleConfig
	solGateway *Gateway
	goGateway  *client.Contract
	startBlock uint64
	logger     log.TMLogger
}

func NewOracle(cfg OracleConfig) *Oracle {
	return &Oracle{cfg: cfg}
}

func (orc *Oracle) Init() error {
	orc.logger = log.Root.With("module", "gateway-oracle")
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

// TODO: Graceful shutdown
func (orc *Oracle) Run() {
	req := &gwc.GatewayStateRequest{}
	callerAddr := loom.RootAddress(orc.cfg.ChainID)
	skipSleep := true
	for {
		if !skipSleep {
			// TODO: should be configurable
			time.Sleep(5 * time.Second)
		} else {
			skipSleep = false
		}

		// TODO: since the oracle is running in-process we can bypass the RPC... but that's going
		// to require a bit of refactoring to avoid duplicating a bunch of QueryServer code... or
		// maybe just pass through an instance of the QueryServer?
		var resp gwc.GatewayStateResponse
		if _, err := orc.goGateway.StaticCall("GetState", req, callerAddr, &resp); err != nil {
			orc.logger.Error("failed to retrieve state from Gateway contract on DAppChain", err)
			continue
		}

		startBlock := resp.State.LastEthBlock + 1
		if orc.startBlock >= startBlock {
			// We've already processed this block successfully... so sit this one out.
			// TODO: figure out if this is actually a good idea
			continue
		}

		batch, lastEthBlock, err := orc.fetchEvents(startBlock)
		if err != nil {
			orc.logger.Error("failed to fetch events from Ethereum", err)
			continue
		}

		if _, err := orc.goGateway.Call("ProcessEventBatch", batch, orc.cfg.Signer, nil); err != nil {
			orc.logger.Error("failed to commit ProcessEventBatch tx", err)
			continue
		}

		orc.startBlock = lastEthBlock + 1
	}
}

func (orc *Oracle) fetchEvents(startBlock uint64) (*gwc.ProcessEventBatchRequest, uint64, error) {
	// NOTE: Currently either all blocks from w.StartBlock are processed successfully or none are.
	lastBlock := startBlock - 1
	ftDeposits := []*gwc.FTDeposit{}
	nftDeposits := []*gwc.NFTDeposit{}

	// TODO: Currently there are 3 separate requests being made, should just make one for all 3 events
	//       because it would be (a) more efficient, and (b) simplify the code a fair bit since
	//       you wouldn't have to track which block range has been processed for each event type.
	ethIt, err := orc.solGateway.FilterETHReceived(&bind.FilterOpts{Start: orc.startBlock})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get logs for ETHReceived")
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
				return nil, 0, errors.Wrap(err, "failed to parse ETHReceived from address")
			}
			ftDeposits = append(ftDeposits, &gwc.FTDeposit{
				Token:    tokenAddr.MarshalPB(),
				From:     loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
				Amount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
				EthBlock: ev.Raw.BlockNumber,
			})
			if lastBlock < ev.Raw.BlockNumber {
				lastBlock = ev.Raw.BlockNumber
			}
		} else {
			err := ethIt.Error()
			if err != nil {
				return nil, 0, errors.Wrap(err, "failed to get event data for ETHReceived")
			}
			ethIt.Close()
			break
		}
	}

	erc20It, err := orc.solGateway.FilterERC20Received(&bind.FilterOpts{Start: orc.startBlock})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get logs for ERC20Received")
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
				return nil, 0, errors.Wrap(err, "failed to parse ERC20Received from address")
			}
			ftDeposits = append(ftDeposits, &gwc.FTDeposit{
				Token:    tokenAddr.MarshalPB(),
				From:     loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
				Amount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
				EthBlock: ev.Raw.BlockNumber,
			})
			if lastBlock < ev.Raw.BlockNumber {
				lastBlock = ev.Raw.BlockNumber
			}
		} else {
			err := erc20It.Error()
			if err != nil {
				return nil, 0, errors.Wrap(err, "failed to get event data for ERC20Received")
			}
			erc20It.Close()
			break
		}
	}

	erc721It, err := orc.solGateway.FilterERC721Received(&bind.FilterOpts{Start: orc.startBlock})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get logs for ERC721Received")
	}
	for {
		ok := erc721It.Next()
		if ok {
			ev := erc721It.Event
			fmt.Printf("ERC721Received: %v from %v in block %v\n",
				ev.Uid.String(), ev.From.Hex(), ev.Raw.BlockNumber)
			localAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, 0, errors.Wrap(err, "failed to parse ERC721Received from address")
			}
			nftDeposits = append(nftDeposits, &gwc.NFTDeposit{
				Token:    loom.RootAddress("eth").MarshalPB(),
				From:     loom.Address{ChainID: "eth", Local: localAddr}.MarshalPB(),
				Uid:      &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Uid)},
				EthBlock: ev.Raw.BlockNumber,
			})
			if lastBlock < ev.Raw.BlockNumber {
				lastBlock = ev.Raw.BlockNumber
			}
		} else {
			err := erc721It.Error()
			if err != nil {
				return nil, 0, errors.Wrap(err, "Failed to get event data for ERC721Received")
			}
			erc721It.Close()
			break
		}
	}

	return &gwc.ProcessEventBatchRequest{
		FtDeposits:  ftDeposits,
		NftDeposits: nftDeposits,
	}, lastBlock, nil
}
