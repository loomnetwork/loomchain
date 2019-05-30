package gateway

import (
	"time"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/client"
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

	ConfirmWithdrawalReceiptRequestV2  = tgtypes.TransferGatewayConfirmWithdrawalReceiptRequestV2
	ClearInvalidDepositTxHashRequest   = tgtypes.TransferGatewayClearInvalidDepositTxHashRequest
	UnprocessedDepositTxHashesResponse = tgtypes.TransferGatewayUnprocessedDepositTxHashesResponse
	UnprocessedDepositTxHashesRequest  = tgtypes.TransferGatewayUnprocessedDepositTxHashesRequest
)

const (
	TokenKind_ERC721X  = tgtypes.TransferGatewayTokenKind_ERC721X
	TokenKind_ERC721   = tgtypes.TransferGatewayTokenKind_ERC721
	TokenKind_ERC20    = tgtypes.TransferGatewayTokenKind_ERC20
	TokenKind_ETH      = tgtypes.TransferGatewayTokenKind_ETH
	TokenKind_LoomCoin = tgtypes.TransferGatewayTokenKind_LOOMCOIN
	TokenKind_TRX      = tgtypes.TransferGatewayTokenKind_TRX
	TokenKind_TRC20    = tgtypes.TransferGatewayTokenKind_TRC20
)

// DAppChainGateway is a partial client-side binding of the Gateway Go contract
type DAppChainGateway struct {
	Address loom.Address
	// Timestamp of the last successful response from the DAppChain
	LastResponseTime time.Time

	contract *client.Contract
	caller   loom.Address
	logger   *loom.Logger
	signer   auth.Signer
}

func ConnectToDAppChainLoomCoinGateway(
	loomClient *client.DAppChainRPCClient, caller loom.Address, signer auth.Signer,
	logger *loom.Logger,
) (*DAppChainGateway, error) {
	gatewayAddr, err := loomClient.Resolve("loomcoin-gateway")
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve Gateway Go contract address")
	}

	return &DAppChainGateway{
		Address:          gatewayAddr,
		LastResponseTime: time.Now(),
		contract:         client.NewContract(loomClient, gatewayAddr.Local),
		caller:           caller,
		signer:           signer,
		logger:           logger,
	}, nil
}

func ConnectToDAppChainGateway(
	loomClient *client.DAppChainRPCClient, caller loom.Address, signer auth.Signer,
	logger *loom.Logger,
) (*DAppChainGateway, error) {
	gatewayAddr, err := loomClient.Resolve("gateway")
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve Gateway Go contract address")
	}

	return &DAppChainGateway{
		Address:          gatewayAddr,
		LastResponseTime: time.Now(),
		contract:         client.NewContract(loomClient, gatewayAddr.Local),
		caller:           caller,
		signer:           signer,
		logger:           logger,
	}, nil
}

func ConnectToDAppChainTronGateway(
	loomClient *client.DAppChainRPCClient, caller loom.Address, signer auth.Signer,
	logger *loom.Logger,
) (*DAppChainGateway, error) {
	gatewayAddr, err := loomClient.Resolve("tron-gateway")
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve Gateway Go contract address")
	}

	return &DAppChainGateway{
		Address:          gatewayAddr,
		LastResponseTime: time.Now(),
		contract:         client.NewContract(loomClient, gatewayAddr.Local),
		caller:           caller,
		signer:           signer,
		logger:           logger,
	}, nil
}

func (gw *DAppChainGateway) LastMainnetBlockNum() (uint64, error) {
	var resp GatewayStateResponse
	if _, err := gw.contract.StaticCall("GetState", &GatewayStateRequest{}, gw.caller, &resp); err != nil {
		gw.logger.Error("failed to retrieve state from Gateway contract on DAppChain", "err", err)
		return 0, err
	}
	gw.LastResponseTime = time.Now()
	return resp.State.LastMainnetBlockNum, nil
}

func (gw *DAppChainGateway) ClearInvalidDepositTxHashes(txHashes [][]byte) error {
	req := &ClearInvalidDepositTxHashRequest{
		TxHashes: txHashes,
	}
	if _, err := gw.contract.Call("ClearInvalidDepositTxHash", req, gw.signer, nil); err != nil {
		gw.logger.Error("failed to commit ClearInvalidLoomCoinDepositTxHash tx", "err", err)
		return err
	}
	gw.LastResponseTime = time.Now()
	return nil
}

func (gw *DAppChainGateway) ProcessDepositEventByTxHash(events []*MainnetEvent) error {
	// TODO: limit max message size to under 1MB
	req := &ProcessEventBatchRequest{
		Events: events,
	}
	if _, err := gw.contract.Call("ProcessDepositEventByTxHash", req, gw.signer, nil); err != nil {
		gw.logger.Error("failed to commit ProcessDepositEventByTxHash tx", "err", err)
		return err
	}
	gw.LastResponseTime = time.Now()
	return nil
}

func (gw *DAppChainGateway) ProcessEventBatch(events []*MainnetEvent) error {
	// TODO: limit max message size to under 1MB
	req := &ProcessEventBatchRequest{
		Events: events,
	}
	if _, err := gw.contract.Call("ProcessEventBatch", req, gw.signer, nil); err != nil {
		gw.logger.Error("failed to commit ProcessEventBatch tx", "err", err)
		return err
	}
	gw.LastResponseTime = time.Now()
	return nil
}

func (gw *DAppChainGateway) PendingWithdrawals(mainnetGatewayAddr loom.Address) ([]*PendingWithdrawalSummary, error) {
	req := &PendingWithdrawalsRequest{
		MainnetGateway: mainnetGatewayAddr.MarshalPB(),
	}
	resp := PendingWithdrawalsResponse{}
	if _, err := gw.contract.StaticCall("PendingWithdrawals", req, gw.caller, &resp); err != nil {
		gw.logger.Error("failed to fetch pending withdrawals from DAppChain", "err", err)
		return nil, err
	}
	gw.LastResponseTime = time.Now()
	return resp.Withdrawals, nil
}

func (gw *DAppChainGateway) UnprocessedLoomCoinDepositTxHash() (*UnprocessedDepositTxHashesResponse, error) {
	req := &UnprocessedDepositTxHashesRequest{}
	resp := UnprocessedDepositTxHashesResponse{}
	if _, err := gw.contract.StaticCall("UnprocessedLoomCoinDepositTxHashes", req, gw.caller, &resp); err != nil {
		gw.logger.Error("failed to fetch unprocessed tx hashesfrom dappchain", "err", err)
		return nil, err
	}
	gw.LastResponseTime = time.Now()
	return &resp, nil
}

func (gw *DAppChainGateway) PendingWithdrawalsV2(mainnetGatewayAddr loom.Address) ([]*PendingWithdrawalSummary, error) {
	req := &PendingWithdrawalsRequest{
		MainnetGateway: mainnetGatewayAddr.MarshalPB(),
	}
	resp := PendingWithdrawalsResponse{}
	if _, err := gw.contract.StaticCall("PendingWithdrawalsV2", req, gw.caller, &resp); err != nil {
		gw.logger.Error("failed to fetch pending withdrawals from DAppChain", "err", err)
		return nil, err
	}
	gw.LastResponseTime = time.Now()
	return resp.Withdrawals, nil
}

func (gw *DAppChainGateway) ConfirmWithdrawalReceipt(req *ConfirmWithdrawalReceiptRequest) error {
	_, err := gw.contract.Call("ConfirmWithdrawalReceipt", req, gw.signer, nil)
	if err != nil {
		return err
	}
	gw.LastResponseTime = time.Now()
	return nil
}

func (gw *DAppChainGateway) ConfirmWithdrawalReceiptV2(req *ConfirmWithdrawalReceiptRequestV2) error {
	_, err := gw.contract.Call("ConfirmWithdrawalReceiptV2", req, gw.signer, nil)
	if err != nil {
		return err
	}
	gw.LastResponseTime = time.Now()
	return nil
}

func (gw *DAppChainGateway) UnverifiedContractCreators() ([]*UnverifiedContractCreator, error) {
	req := &UnverifiedContractCreatorsRequest{}
	resp := UnverifiedContractCreatorsResponse{}
	if _, err := gw.contract.StaticCall("UnverifiedContractCreators", req, gw.caller, &resp); err != nil {
		gw.logger.Error("failed to fetch pending contract mappings from DAppChain", "err", err)
		return nil, err
	}
	gw.LastResponseTime = time.Now()
	return resp.Creators, nil
}

func (gw *DAppChainGateway) VerifyContractCreators(verifiedCreators []*VerifiedContractCreator) error {
	req := &VerifyContractCreatorsRequest{
		Creators: verifiedCreators,
	}
	_, err := gw.contract.Call("VerifyContractCreators", req, gw.signer, nil)
	if err != nil {
		return err
	}
	gw.LastResponseTime = time.Now()
	return nil
}
