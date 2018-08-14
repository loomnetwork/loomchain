// +build evm

package gateway

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/gateway/ethcontract"
	"github.com/pkg/errors"
)

type (
	ProcessEventBatchRequest           = tgtypes.TransferGatewayProcessEventBatchRequest
	GatewayStateRequest                = tgtypes.TransferGatewayStateRequest
	GatewayStateResponse               = tgtypes.TransferGatewayStateResponse
	ConfirmWithdrawalReceiptRequest    = tgtypes.TransferGatewayConfirmWithdrawalReceiptRequest
	PendingWithdrawalsRequest          = tgtypes.TransferGatewayPendingWithdrawalsRequest
	PendingWithdrawalsResponse         = tgtypes.TransferGatewayPendingWithdrawalsResponse
	MainnetEvent                       = tgtypes.TransferGatewayMainnetEvent
	MainnetDepositEvent                = tgtypes.TransferGatewayMainnetEvent_Deposit
	MainnetWithdrawalEvent             = tgtypes.TransferGatewayMainnetEvent_Withdrawal
	MainnetTokenDeposited              = tgtypes.TransferGatewayTokenDeposited
	MainnetTokenWithdrawn              = tgtypes.TransferGatewayTokenWithdrawn
	TokenKind                          = tgtypes.TransferGatewayTokenKind
	PendingWithdrawalSummary           = tgtypes.TransferGatewayPendingWithdrawalSummary
	UnverifiedContractCreatorsRequest  = tgtypes.TransferGatewayUnverifiedContractCreatorsRequest
	UnverifiedContractCreatorsResponse = tgtypes.TransferGatewayUnverifiedContractCreatorsResponse
	VerifyContractCreatorsRequest      = tgtypes.TransferGatewayVerifyContractCreatorsRequest
	UnverifiedContractCreator          = tgtypes.TransferGatewayUnverifiedContractCreator
	VerifiedContractCreator            = tgtypes.TransferGatewayVerifiedContractCreator
)

const (
	TokenKind_ERC721 = tgtypes.TransferGatewayTokenKind_ERC721
)

type mainnetEventInfo struct {
	BlockNum uint64
	TxIdx    uint
	Event    *MainnetEvent
}

type Oracle struct {
	cfg        TransferGatewayConfig
	chainID    string
	solGateway *ethcontract.MainnetGatewayContract
	goGateway  *client.Contract
	startBlock uint64
	logger     *loom.Logger
	ethClient  *MainnetClient
	address    loom.Address
	// Used to sign tx/data sent to the DAppChain Gateway contract
	signer auth.Signer
	// Private key that should be used to sign tx/data sent to Mainnet Gateway contract
	mainnetPrivateKey     *ecdsa.PrivateKey
	dAppChainPollInterval time.Duration
	mainnetPollInterval   time.Duration
	startupDelay          time.Duration
	reconnectInterval     time.Duration
}

func CreateOracle(cfg *TransferGatewayConfig, chainID string) (*Oracle, error) {
	privKey, err := LoadDAppChainPrivateKey(cfg.DAppChainPrivateKeyPath)
	if err != nil {
		return nil, err
	}
	signer := auth.NewEd25519Signer(privKey)

	mainnetPrivateKey, err := LoadMainnetPrivateKey(cfg.MainnetPrivateKeyPath)
	if err != nil {
		return nil, err
	}

	return &Oracle{
		cfg:     *cfg,
		chainID: chainID,
		logger:  loom.NewLoomLogger(cfg.OracleLogLevel, cfg.OracleLogDestination),
		address: loom.Address{
			ChainID: chainID,
			Local:   loom.LocalAddressFromPublicKey(signer.PublicKey()),
		},
		signer:                signer,
		mainnetPrivateKey:     mainnetPrivateKey,
		dAppChainPollInterval: time.Duration(cfg.DAppChainPollInterval) * time.Second,
		mainnetPollInterval:   time.Duration(cfg.MainnetPollInterval) * time.Second,
		startupDelay:          time.Duration(cfg.OracleStartupDelay) * time.Second,
		reconnectInterval:     time.Duration(cfg.OracleReconnectInterval) * time.Second,
	}, nil
}

func (orc *Oracle) connect() error {
	var err error
	orc.ethClient, err = ConnectToMainnet(orc.cfg.EthereumURI)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Ethereum")
	}

	orc.solGateway, err = ethcontract.NewMainnetGatewayContract(
		common.HexToAddress(orc.cfg.MainnetContractHexAddress), orc.ethClient)
	if err != nil {
		return errors.Wrap(err, "failed to bind Gateway Solidity contract")
	}

	dappClient := client.NewDAppChainRPCClient(orc.chainID, orc.cfg.DAppChainWriteURI, orc.cfg.DAppChainReadURI)
	contractAddr, err := dappClient.Resolve("gateway")
	if err != nil {
		return errors.Wrap(err, "failed to resolve Gateway Go contract address")
	}
	orc.goGateway = client.NewContract(dappClient, contractAddr.Local)
	return nil
}

// RunWithRecovery should run in a goroutine, it will ensure the oracle keeps on running as long
// as it doesn't panic due to a runtime error.
func (orc *Oracle) RunWithRecovery() {
	defer func() {
		if r := recover(); r != nil {
			orc.logger.Error("recovered from panic in Gateway Oracle", "r", r)
			// Unless it's a runtime error restart the goroutine
			if _, ok := r.(runtime.Error); !ok {
				time.Sleep(30 * time.Second)
				orc.logger.Info("Restarting Gateway Oracle...")
				go orc.RunWithRecovery()
			}
		}
	}()

	// When running in-process give the node a bit of time to spin up.
	if orc.startupDelay > 0 {
		time.Sleep(orc.startupDelay)
	}

	orc.Run()
}

// TODO: Graceful shutdown
func (orc *Oracle) Run() {
	for {
		if err := orc.connect(); err == nil {
			break
		}
		time.Sleep(orc.reconnectInterval)
	}

	skipSleep := true
	for {
		if !skipSleep {
			time.Sleep(orc.mainnetPollInterval)
		} else {
			skipSleep = false
		}
		// TODO: should be possible to poll DAppChain & Mainnet at different intervals
		orc.pollMainnet()
		orc.pollDAppChain()
	}
}

func (orc *Oracle) pollMainnet() error {
	req := &GatewayStateRequest{}
	// TODO: If the oracle is running in-process we could probably bypass the RPC interface for
	//       static calls.
	var resp GatewayStateResponse
	if _, err := orc.goGateway.StaticCall("GetState", req, orc.address, &resp); err != nil {
		orc.logger.Error("failed to retrieve state from Gateway contract on DAppChain", "err", err)
		return err
	}

	startBlock := resp.State.LastMainnetBlockNum + 1
	if orc.startBlock > startBlock {
		startBlock = orc.startBlock
	}

	// TODO: limit max block range per batch
	latestBlock, err := orc.getLatestEthBlockNumber()
	if err != nil {
		orc.logger.Error("failed to obtain latest Ethereum block number", "err", err)
		return err
	}

	if latestBlock < startBlock {
		// Wait for Ethereum to produce a new block...
		return nil
	}

	batch, err := orc.fetchEvents(startBlock, latestBlock)
	if err != nil {
		orc.logger.Error("failed to fetch events from Ethereum", "err", err)
		return err
	}

	if len(batch.Events) > 0 {
		if _, err := orc.goGateway.Call("ProcessEventBatch", batch, orc.signer, nil); err != nil {
			orc.logger.Error("failed to commit ProcessEventBatch tx", "err", err)
			return err
		}
	}

	orc.startBlock = latestBlock + 1
	return nil
}

func (orc *Oracle) pollDAppChain() error {
	if err := orc.verifyContractCreators(); err != nil {
		return err
	}

	// TODO: should probably just log errors and soldier on
	if err := orc.signPendingWithdrawals(); err != nil {
		return err
	}
	return nil
}

// TODO: Need some way of keeping track which withdrawals the oracle has signed already because there
//       may be a delay before the node state is updated, so it's possible for the oracle to retrieve,
//       sign, and resubmit withdrawals it has already signed.
func (orc *Oracle) signPendingWithdrawals() error {
	req := &PendingWithdrawalsRequest{}
	resp := PendingWithdrawalsResponse{}
	if _, err := orc.goGateway.StaticCall("PendingWithdrawals", req, orc.address, &resp); err != nil {
		orc.logger.Error("failed to fetch pending withdrawals from DAppChain", "err", err)
		return err
	}

	for _, summary := range resp.Withdrawals {
		sig, err := orc.signTransferGatewayWithdrawal(summary.Hash)
		if err != nil {
			return err
		}
		req := &ConfirmWithdrawalReceiptRequest{
			TokenOwner:      summary.TokenOwner,
			OracleSignature: sig,
			WithdrawalHash:  summary.Hash,
		}
		_, err = orc.goGateway.Call("ConfirmWithdrawalReceipt", req, orc.signer, nil)
		// Ignore errors indicating a receipt has been signed already, they simply indicate another
		// Oracle has managed to sign the receipt already.
		// TODO: replace hardcoded error message with gateway.ErrWithdrawalReceiptSigned when this
		//       code is moved back into loomchain
		if err != nil {
			if strings.HasPrefix(err.Error(), "TG006:") {
				orc.logger.Debug("withdrawal already signed",
					"tokenOwner", loom.UnmarshalAddressPB(summary.TokenOwner).String(),
					"hash", hex.EncodeToString(summary.Hash),
				)
			} else {
				return err
			}
		}
		orc.logger.Debug("submitted signed withdrawal to DAppChain",
			"tokenOwner", loom.UnmarshalAddressPB(summary.TokenOwner).String(),
			"hash", hex.EncodeToString(summary.Hash),
		)
	}
	return nil
}

func (orc *Oracle) verifyContractCreators() error {
	unverifiedReq := &UnverifiedContractCreatorsRequest{}
	unverifiedResp := UnverifiedContractCreatorsResponse{}
	if _, err := orc.goGateway.StaticCall("UnverifiedContractCreators", unverifiedReq, orc.address, &unverifiedResp); err != nil {
		orc.logger.Error("failed to fetch pending contract mappings from DAppChain", "err", err)
		return err
	}

	if len(unverifiedResp.Creators) == 0 {
		return nil
	}

	verifiedCreators := make([]*VerifiedContractCreator, 0, len(unverifiedResp.Creators))
	for _, unverifiedCreator := range unverifiedResp.Creators {
		verifiedCreator, err := orc.fetchMainnetContractCreator(unverifiedCreator)
		if err != nil {
			orc.logger.Debug("failed to fetch Mainnet contract creator", "err", err)
		} else {
			verifiedCreators = append(verifiedCreators, verifiedCreator)
		}
	}

	verifiedReq := &VerifyContractCreatorsRequest{
		Creators: verifiedCreators,
	}
	_, err := orc.goGateway.Call("VerifyContractCreators", verifiedReq, orc.signer, nil)
	return err
}

func (orc *Oracle) fetchMainnetContractCreator(unverified *UnverifiedContractCreator) (*VerifiedContractCreator, error) {
	verifiedCreator := &VerifiedContractCreator{
		ContractMappingID: unverified.ContractMappingID,
		Creator:           loom.RootAddress("eth").MarshalPB(),
		Contract:          loom.RootAddress("eth").MarshalPB(),
	}
	txHash := common.BytesToHash(unverified.ContractTxHash)
	tx, err := orc.ethClient.ContractCreationTxByHash(context.TODO(), txHash)
	if err == ethereum.NotFound {
		return verifiedCreator, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to find contract creator by tx hash %v", txHash)
	}
	verifiedCreator.Creator.Local = loom.LocalAddress(tx.CreatorAddress.Bytes())
	verifiedCreator.Contract.Local = loom.LocalAddress(tx.ContractAddress.Bytes())
	return verifiedCreator, nil
}

func (orc *Oracle) getLatestEthBlockNumber() (uint64, error) {
	blockHeader, err := orc.ethClient.HeaderByNumber(context.TODO(), nil)
	if err != nil {
		return 0, err
	}
	return blockHeader.Number.Uint64(), nil
}

// Fetches all relevent events from an Ethereum node from startBlock to endBlock (inclusive)
func (orc *Oracle) fetchEvents(startBlock, endBlock uint64) (*ProcessEventBatchRequest, error) {
	// NOTE: Currently either all blocks from w.StartBlock are processed successfully or none are.
	filterOpts := &bind.FilterOpts{
		Start: startBlock,
		End:   &endBlock,
	}

	deposits, err := orc.fetchERC721Deposits(filterOpts)
	if err != nil {
		return nil, err
	}

	withdrawals, err := orc.fetchTokenWithdrawals(filterOpts)
	if err != nil {
		return nil, err
	}

	events := append(deposits, withdrawals...)
	sortMainnetEvents(events)
	sortedEvents := make([]*MainnetEvent, len(events))
	for i, event := range events {
		sortedEvents[i] = event.Event
	}

	if len(events) > 0 {
		orc.logger.Debug("fetched Mainnet events",
			"startBlock", startBlock,
			"endBlock", endBlock,
			"deposits", len(deposits),
			"withdrawals", len(withdrawals),
		)
	}

	return &ProcessEventBatchRequest{
		Events: sortedEvents,
	}, nil
}

func sortMainnetEvents(events []*mainnetEventInfo) {
	// Sort events by block & tx index (within the block)
	sort.Slice(events, func(i, j int) bool {
		if events[i].BlockNum == events[j].BlockNum {
			return events[i].TxIdx < events[j].TxIdx
		}
		return events[i].BlockNum < events[j].BlockNum
	})
}

func (orc *Oracle) fetchERC721Deposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	erc721It, err := orc.solGateway.FilterERC721Received(filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ERC721Received")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := erc721It.Next()
		if ok {
			ev := erc721It.Event
			tokenAddr, err := loom.LocalAddressFromHexString(ev.ContractAddress.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721Received token address")
			}
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721Received from address")
			}
			events = append(events, &mainnetEventInfo{
				BlockNum: ev.Raw.BlockNumber,
				TxIdx:    ev.Raw.TxIndex,
				Event: &MainnetEvent{
					EthBlock: ev.Raw.BlockNumber,
					Payload: &MainnetDepositEvent{
						Deposit: &MainnetTokenDeposited{
							TokenKind:     TokenKind_ERC721,
							TokenContract: loom.Address{ChainID: "eth", Local: tokenAddr}.MarshalPB(),
							TokenOwner:    loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
							Value:         &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Uid)},
						},
					},
				},
			})
		} else {
			err := erc721It.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for ERC721Received")
			}
			erc721It.Close()
			break
		}
	}
	return events, nil
}

func (orc *Oracle) fetchTokenWithdrawals(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	it, err := orc.solGateway.FilterTokenWithdrawn(filterOpts, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ERC721Received")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := it.Next()
		if ok {
			ev := it.Event
			tokenAddr, err := loom.LocalAddressFromHexString(ev.ContractAddress.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721Received token address")
			}
			fromAddr, err := loom.LocalAddressFromHexString(ev.Owner.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721Received from address")
			}
			events = append(events, &mainnetEventInfo{
				BlockNum: ev.Raw.BlockNumber,
				TxIdx:    ev.Raw.TxIndex,
				Event: &MainnetEvent{
					EthBlock: ev.Raw.BlockNumber,
					Payload: &MainnetWithdrawalEvent{
						Withdrawal: &MainnetTokenWithdrawn{
							TokenKind:     TokenKind(ev.Kind),
							TokenContract: loom.Address{ChainID: "eth", Local: tokenAddr}.MarshalPB(),
							TokenOwner:    loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
							Value:         &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Value)},
						},
					},
				},
			})
		} else {
			err := it.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for ERC721Received")
			}
			it.Close()
			break
		}
	}
	return events, nil
}

func (orc *Oracle) signTransferGatewayWithdrawal(hash []byte) ([]byte, error) {
	sig, err := evmcompat.SoliditySign(hash, orc.mainnetPrivateKey)
	if err != nil {
		return nil, err
	}
	// The first byte should be the signature mode, for details about the signature format refer to
	// https://github.com/loomnetwork/plasma-erc721/blob/master/server/contracts/Libraries/ECVerify.sol
	return append(make([]byte, 1, 66), sig...), nil
}

func LoadDAppChainPrivateKey(path string) ([]byte, error) {
	privKeyB64, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privKey, err := base64.StdEncoding.DecodeString(string(privKeyB64))
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

func LoadMainnetPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	privKey, err := crypto.LoadECDSA(path)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}
