// +build evm

package gateway

import (
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-ethereum/accounts/abi"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/client/erc20"
	lcrypto "github.com/loomnetwork/go-loom/crypto"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	gwcontract "github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	"github.com/loomnetwork/loomchain/gateway/ethcontract"
	"github.com/pkg/errors"
)

type recentHashPool struct {
	hashMap         map[string]bool
	cleanupInterval time.Duration
	ticker          *time.Ticker
	stopCh          chan struct{}

	accessMutex sync.RWMutex
}

func newRecentHashPool(cleanupInterval time.Duration) *recentHashPool {
	return &recentHashPool{
		hashMap:         make(map[string]bool),
		cleanupInterval: cleanupInterval,
	}
}

func (r *recentHashPool) addHash(hash []byte) bool {
	r.accessMutex.Lock()
	defer r.accessMutex.Unlock()

	hexEncodedHash := hex.EncodeToString(hash)

	if _, ok := r.hashMap[hexEncodedHash]; ok {
		// If we are returning false, this means we have already seen hash
		return false
	}

	r.hashMap[hexEncodedHash] = true
	return true
}

func (r *recentHashPool) seenHash(hash []byte) bool {
	r.accessMutex.RLock()
	defer r.accessMutex.RUnlock()

	hexEncodedHash := hex.EncodeToString(hash)

	_, ok := r.hashMap[hexEncodedHash]
	return ok
}

func (r *recentHashPool) startCleanupRoutine() {
	r.ticker = time.NewTicker(r.cleanupInterval)
	r.stopCh = make(chan struct{})

	go func() {
		for {
			select {
			case <-r.stopCh:
				return
			case <-r.ticker.C:
				r.accessMutex.Lock()
				r.hashMap = make(map[string]bool)
				r.accessMutex.Unlock()
				break
			}
		}
	}()

}

func (r *recentHashPool) stopCleanupRoutine() {
	close(r.stopCh)
	r.ticker.Stop()
}

type mainnetEventInfo struct {
	BlockNum uint64
	TxIdx    uint
	Event    *MainnetEvent
}

type Status struct {
	Version                  string
	OracleAddress            string
	DAppChainGatewayAddress  string
	MainnetGatewayAddress    string
	NextMainnetBlockNum      uint64    `json:",string"`
	MainnetGatewayLastSeen   time.Time // TODO: hook this up
	DAppChainGatewayLastSeen time.Time
	// Number of Mainnet events submitted to the DAppChain Gateway successfully
	NumMainnetEventsFetched uint64 `json:",string"`
	// Total number of Mainnet events fetched
	NumMainnetEventsSubmitted uint64 `json:",string"`
}

type Oracle struct {
	cfg        TransferGatewayConfig
	chainID    string
	solGateway *ethcontract.MainnetGatewayContract
	goGateway  *DAppChainGateway
	startBlock uint64
	logger     *loom.Logger
	ethClient  *MainnetClient
	address    loom.Address
	// Used to sign tx/data sent to the DAppChain Gateway contract
	signer auth.Signer
	// Private key that should be used to sign tx/data sent to Mainnet Gateway contract
	mainnetPrivateKey     lcrypto.PrivateKey
	dAppChainPollInterval time.Duration
	mainnetPollInterval   time.Duration
	startupDelay          time.Duration
	reconnectInterval     time.Duration
	mainnetGatewayAddress loom.Address

	numMainnetBlockConfirmations uint64
	numMainnetEventsFetched      uint64
	numMainnetEventsSubmitted    uint64

	statusMutex sync.RWMutex
	status      Status

	metrics *Metrics

	hashPool *recentHashPool

	gatewayType         gwcontract.GatewayType
	withdrawalSig       WithdrawalSigType
	withdrawerBlacklist []loom.Address
	erc20ABI            abi.ABI

	receiptSigningEnabled bool

	// Tron specific
	tronClient *TronClient
}

func CreateOracle(cfg *TransferGatewayConfig, chainID string) (*Oracle, error) {
	return createOracle(cfg, chainID, "tg_oracle", gwcontract.EthereumGateway, "eth")
}

func CreateLoomCoinOracle(cfg *TransferGatewayConfig, chainID string) (*Oracle, error) {
	return createOracle(cfg, chainID, "loom_tg_oracle", gwcontract.LoomCoinGateway, "eth")
}

func CreateTronOracle(cfg *TransferGatewayConfig, chainID string) (*Oracle, error) {
	return createOracle(cfg, chainID, "tron_tg_oracle", gwcontract.TronGateway, "tron")
}

func createOracle(cfg *TransferGatewayConfig, chainID string,
	metricSubsystem string, gatewayType gwcontract.GatewayType,
	foreignChainID string,
) (*Oracle, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var signerType string

	privKey, err := LoadDAppChainPrivateKey(cfg.DappChainPrivateKeyHsmEnabled, cfg.DAppChainPrivateKeyPath)
	if err != nil {
		return nil, err
	}

	if cfg.DappChainPrivateKeyHsmEnabled {
		signerType = auth.SignerTypeYubiHsm
	} else {
		signerType = auth.SignerTypeEd25519
	}
	signer := auth.NewSigner(signerType, privKey)

	receiptSigningEnabled := !cfg.BatchSignFnConfig.Enabled

	// only load the mainnet private key if receipt signing is needed
	var mainnetPrivateKey lcrypto.PrivateKey
	if receiptSigningEnabled {
		mainnetPrivateKey, err = LoadMainnetPrivateKey(cfg.MainnetPrivateKeyHsmEnabled, cfg.MainnetPrivateKeyPath)
		if err != nil {
			return nil, err

		}
	}

	address := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(signer.PublicKey()),
	}

	if !common.IsHexAddress(cfg.MainnetContractHexAddress) {
		return nil, errors.New("invalid Mainnet Gateway address")
	}

	withdrawerBlacklist, err := cfg.GetWithdrawerAddressBlacklist()
	if err != nil {
		return nil, err
	}

	hashPool := newRecentHashPool(time.Duration(cfg.MainnetPollInterval) * time.Second * 4)
	hashPool.startCleanupRoutine()

	erc20ABI, err := abi.JSON(strings.NewReader(erc20.ERC20ABI))
	if err != nil {
		return nil, err
	}

	return &Oracle{
		cfg:                          *cfg,
		chainID:                      chainID,
		logger:                       loom.NewLoomLogger(cfg.OracleLogLevel, cfg.OracleLogDestination),
		address:                      address,
		signer:                       signer,
		mainnetPrivateKey:            mainnetPrivateKey,
		dAppChainPollInterval:        time.Duration(cfg.DAppChainPollInterval) * time.Second,
		mainnetPollInterval:          time.Duration(cfg.MainnetPollInterval) * time.Second,
		numMainnetBlockConfirmations: uint64(cfg.NumMainnetBlockConfirmations),
		startupDelay:                 time.Duration(cfg.OracleStartupDelay) * time.Second,
		reconnectInterval:            time.Duration(cfg.OracleReconnectInterval) * time.Second,
		mainnetGatewayAddress: loom.Address{
			ChainID: foreignChainID,
			Local:   common.HexToAddress(cfg.MainnetContractHexAddress).Bytes(),
		},
		status: Status{
			Version:               loomchain.FullVersion(),
			OracleAddress:         address.String(),
			MainnetGatewayAddress: cfg.MainnetContractHexAddress,
		},

		metrics:             NewMetrics(metricSubsystem),
		hashPool:            hashPool,
		gatewayType:         gatewayType,
		withdrawalSig:       cfg.WithdrawalSig,
		withdrawerBlacklist: withdrawerBlacklist,
		// Oracle will do receipt signing when BatchSignFnConfig is disabled
		receiptSigningEnabled: receiptSigningEnabled,

		erc20ABI: erc20ABI,
	}, nil
}

// Status returns some basic info about the current state of the Oracle.
func (orc *Oracle) Status() *Status {
	orc.statusMutex.RLock()

	s := orc.status

	orc.statusMutex.RUnlock()
	return &s
}

func (orc *Oracle) updateStatus() {
	orc.statusMutex.Lock()

	orc.status.NextMainnetBlockNum = orc.startBlock
	orc.status.NumMainnetEventsFetched = orc.numMainnetEventsFetched
	orc.status.NumMainnetEventsSubmitted = orc.numMainnetEventsSubmitted

	if orc.goGateway != nil {
		orc.status.DAppChainGatewayAddress = orc.goGateway.Address.String()
		orc.status.DAppChainGatewayLastSeen = orc.goGateway.LastResponseTime
	}

	orc.statusMutex.Unlock()
}

func (orc *Oracle) connect() error {
	var err error
	switch orc.gatewayType {
	case gwcontract.EthereumGateway, gwcontract.LoomCoinGateway:
		if orc.ethClient == nil {
			orc.ethClient, err = ConnectToMainnet(orc.cfg.EthereumURI)
			if err != nil {
				return errors.Wrap(err, "failed to connect to Ethereum mainnet network")
			}
		}

		if orc.solGateway == nil {
			orc.solGateway, err = ethcontract.NewMainnetGatewayContract(
				common.HexToAddress(orc.cfg.MainnetContractHexAddress),
				orc.ethClient,
			)
			if err != nil {
				return errors.Wrap(err, "failed create Mainnet Gateway contract binding")
			}
		}
	case gwcontract.TronGateway:
		if orc.tronClient == nil {
			oracleEventPollDelay := time.Second * time.Duration(orc.cfg.OracleEventPollDelay)
			orc.tronClient, err = ConnectToTron(orc.cfg.TronURI, oracleEventPollDelay)
			if err != nil {
				return errors.Wrap(err, "failed to connect to Tron mainnet network")
			}
		}
	default:
		return errors.Errorf("invalid gateway type %v", orc.gatewayType)
	}

	if orc.goGateway == nil {
		dappClient := client.NewDAppChainRPCClient(orc.chainID, orc.cfg.DAppChainWriteURI, orc.cfg.DAppChainReadURI)
		switch orc.gatewayType {
		case gwcontract.EthereumGateway:
			orc.goGateway, err = ConnectToDAppChainGateway(dappClient, orc.address, orc.signer, orc.logger)
			if err != nil {
				return errors.Wrap(err, "failed to create dappchain gateway")
			}
		case gwcontract.LoomCoinGateway:
			orc.goGateway, err = ConnectToDAppChainLoomCoinGateway(dappClient, orc.address, orc.signer, orc.logger)
			if err != nil {
				return errors.Wrap(err, "failed to create dappchain loomcoin gateway")
			}
		case gwcontract.TronGateway:
			orc.goGateway, err = ConnectToDAppChainTronGateway(dappClient, orc.address, orc.signer, orc.logger)
			if err != nil {
				return errors.Wrap(err, "failed to create dappchain tron gateway")
			}
		default:
			return errors.Errorf("invalid gateway type %v", orc.gatewayType)
		}
	}
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
		if err := orc.connect(); err != nil {
			orc.logger.Error("[TG Oracle] failed to connect", "err", err)
			orc.updateStatus()
		} else {
			orc.updateStatus()
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
	lastMainnetBlockNum, err := orc.goGateway.LastMainnetBlockNum()
	if err != nil {
		return err
	}

	startBlock := lastMainnetBlockNum + 1
	if orc.startBlock > startBlock {
		startBlock = orc.startBlock
	}

	// TODO: limit max block range per batch
	latestBlock, err := orc.getLatestEthBlockNumber()
	if err != nil {
		orc.logger.Error("failed to obtain latest Ethereum block number", "err", err)
		return err
	}

	// Don't process a block until it's been confirmed
	if latestBlock <= orc.numMainnetBlockConfirmations {
		return nil
	}
	latestBlock -= orc.numMainnetBlockConfirmations

	if latestBlock < startBlock {
		// Wait for Ethereum to produce a new block...
		return nil
	}

	events, err := orc.fetchEvents(startBlock, latestBlock)
	if err != nil {
		orc.logger.Error("failed to fetch events from Ethereum", "err", err)
		return err
	}

	solLoomAddr, err := orc.solGateway.LoomAddress(&bind.CallOpts{
		Context: context.TODO(),
	})
	if err != nil {
		return err
	}

	if err := orc.processEventsByTxHash(solLoomAddr); err != nil {
		return err
	}

	if len(events) > 0 {
		orc.numMainnetEventsFetched = orc.numMainnetEventsFetched + uint64(len(events))
		orc.updateStatus()

		if err := orc.goGateway.ProcessEventBatch(events); err != nil {
			return err
		}

		orc.numMainnetEventsSubmitted = orc.numMainnetEventsSubmitted + uint64(len(events))
		orc.metrics.SubmittedMainnetEvents(len(events))
		orc.updateStatus()
	}

	orc.startBlock = latestBlock + 1
	return nil
}

func (orc *Oracle) pollDAppChain() error {
	if err := orc.verifyContractCreators(); err != nil {
		return err
	}

	if orc.receiptSigningEnabled {
		// TODO: should probably just log errors and soldier on
		if err := orc.signPendingWithdrawals(); err != nil {
			return err
		}
	}

	return nil
}

func (orc *Oracle) getLoomCoinDepositFromTxHash(solLoomAddr common.Address) ([]*MainnetEvent, [][]byte, error) {
	unprocessedTxHashesResponse, err := orc.goGateway.UnprocessedLoomCoinDepositTxHash()
	if err != nil {
		return nil, nil, err
	}

	unprocessedTxHashes := unprocessedTxHashesResponse.TxHashes

	mainnetEvents := make([]*MainnetEvent, 0, len(unprocessedTxHashes))
	invalidTxHashes := make([][]byte, 0, len(unprocessedTxHashes))

	for _, unprocessedTxHash := range unprocessedTxHashes {
		erc20Deposits, err := orc.ethClient.GetERC20DepositByTxHash(context.TODO(), orc.erc20ABI, solLoomAddr, common.BytesToHash(unprocessedTxHash))
		if err != nil {
			if err == ethereum.NotFound {
				invalidTxHashes = append(invalidTxHashes, unprocessedTxHash)
				continue
			}
			return nil, nil, err
		}

		if len(erc20Deposits) == 0 {
			invalidTxHashes = append(invalidTxHashes, unprocessedTxHash)
			continue
		}

		// In gateway contract only one deposit can be associated with one tx hash
		// So, let's only take first element of deposit array.
		erc20Deposit := erc20Deposits[0]

		mainnetEvents = append(mainnetEvents, &MainnetEvent{
			EthBlock: erc20Deposit.BlockNumber,
			Payload: &MainnetDepositEvent{
				Deposit: &MainnetTokenDeposited{
					TokenKind:   TokenKind_LoomCoin,
					TokenOwner:  loom.Address{ChainID: "eth", Local: erc20Deposit.From.Bytes()}.MarshalPB(),
					TokenAmount: &ltypes.BigUInt{Value: *loom.NewBigUInt(erc20Deposit.Amount)},
					TxHash:      unprocessedTxHash,
				},
			},
		})
	}

	return mainnetEvents, invalidTxHashes, nil
}

func (orc *Oracle) filterSeenWithdrawals(withdrawals []*PendingWithdrawalSummary) []*PendingWithdrawalSummary {
	unseenWithdrawals := make([]*PendingWithdrawalSummary, len(withdrawals))

	currentIndex := 0
	for _, withdrawal := range withdrawals {
		if !orc.hashPool.addHash(withdrawal.Hash) {
			continue
		}

		unseenWithdrawals[currentIndex] = withdrawal
		currentIndex++
	}

	return unseenWithdrawals[:currentIndex]
}

func (orc *Oracle) signPendingWithdrawals() error {
	var err error
	var numWithdrawalsSigned int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "signPendingWithdrawals", err)
		orc.metrics.WithdrawalsSigned(numWithdrawalsSigned)
		orc.updateStatus()
	}(time.Now())

	var withdrawals []*PendingWithdrawalSummary
	if orc.withdrawalSig == UnprefixedWithdrawalSigType { // old gateway
		withdrawals, err = orc.goGateway.PendingWithdrawals(orc.mainnetGatewayAddress)
	} else { // new gateway
		withdrawals, err = orc.goGateway.PendingWithdrawalsV2(orc.mainnetGatewayAddress)
	}

	if err != nil {
		return err
	}

	// Filter already seen withdrawals in 4 * pollInterval time
	filteredWithdrawals := orc.filterSeenWithdrawals(withdrawals)

	for _, summary := range filteredWithdrawals {
		tokenOwner := loom.UnmarshalAddressPB(summary.TokenOwner)

		skipWithdrawal := false
		for i := range orc.withdrawerBlacklist {
			if orc.withdrawerBlacklist[i].Compare(tokenOwner) == 0 {
				orc.logger.Info(
					"Withdrawer is blacklisted, won't sign withdrawal",
					"tokenOwner", tokenOwner.String(),
					"hash", hex.EncodeToString(summary.Hash),
				)
				skipWithdrawal = true
				break
			}
		}

		if skipWithdrawal {
			continue
		}

		sig, err := orc.signTransferGatewayWithdrawal(summary.Hash)
		if err != nil {
			return err
		}
		req := &ConfirmWithdrawalReceiptRequest{
			TokenOwner:      summary.TokenOwner,
			OracleSignature: sig,
			WithdrawalHash:  summary.Hash,
		}
		// Ignore errors indicating a receipt has been signed already, they simply indicate another
		// Oracle has managed to sign the receipt already.
		// TODO: replace hardcoded error message with gateway.ErrWithdrawalReceiptSigned when this
		//       code is moved back into loomchain
		if err = orc.goGateway.ConfirmWithdrawalReceipt(req); err != nil {
			if strings.HasPrefix(err.Error(), "TG006:") {
				orc.logger.Debug("withdrawal already signed",
					"tokenOwner", tokenOwner.String(),
					"hash", hex.EncodeToString(summary.Hash),
				)
				err = nil
			} else {
				return err
			}
		} else {
			numWithdrawalsSigned++
			orc.logger.Debug("submitted signed withdrawal to DAppChain",
				"tokenOwner", tokenOwner.String(),
				"hash", hex.EncodeToString(summary.Hash),
			)
		}
	}
	return nil
}

func (orc *Oracle) verifyContractCreators() error {
	var err error
	var numContractCreatorsVerified int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "verifyContractCreators", err)
		orc.metrics.ContractCreatorsVerified(numContractCreatorsVerified)
		orc.updateStatus()
	}(time.Now())

	unverifiedCreators, err := orc.goGateway.UnverifiedContractCreators()
	if err != nil {
		return err
	}

	if len(unverifiedCreators) == 0 {
		return nil
	}

	verifiedCreators := make([]*VerifiedContractCreator, 0, len(unverifiedCreators))
	for _, unverifiedCreator := range unverifiedCreators {
		verifiedCreator, err := orc.fetchMainnetContractCreator(unverifiedCreator)
		if err != nil {
			orc.logger.Debug("failed to fetch Mainnet contract creator", "err", err)
		} else {
			verifiedCreators = append(verifiedCreators, verifiedCreator)
			numContractCreatorsVerified++
		}
	}

	err = orc.goGateway.VerifyContractCreators(verifiedCreators)
	return err
}

func (orc *Oracle) fetchMainnetContractCreator(unverified *UnverifiedContractCreator) (*VerifiedContractCreator, error) {
	switch orc.gatewayType {
	case gwcontract.EthereumGateway, gwcontract.LoomCoinGateway:
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

	case gwcontract.TronGateway:
		verifiedCreator := &VerifiedContractCreator{
			ContractMappingID: unverified.ContractMappingID,
			Creator:           loom.RootAddress("tron").MarshalPB(),
			Contract:          unverified.ContractAddress,
		}
		// Got to prefix with 41 instead of 0x
		var address = fmt.Sprintf("41%s", unverified.ContractAddress.Local.Hex())
		gwContract, err := orc.tronClient.GetContract(context.TODO(), address)
		if err == ErrTronContractNotFound {
			return verifiedCreator, nil
		} else if err != nil {
			return nil, err
		}

		creatorAddress := strings.TrimPrefix(gwContract.OriginalAddress, "41")
		creatorAddress = fmt.Sprintf("0x%s", creatorAddress)
		creatorLocalAddress, err := loom.LocalAddressFromHexString(creatorAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert contract creator address %s", creatorAddress)
		}
		verifiedCreator.Creator.Local = creatorLocalAddress

		return verifiedCreator, nil
	}
	return nil, errors.Errorf("invalid gateway type %v", orc.gatewayType)
}

func (orc *Oracle) getLatestEthBlockNumber() (uint64, error) {
	switch orc.gatewayType {
	case gwcontract.EthereumGateway, gwcontract.LoomCoinGateway:
		blockHeader, err := orc.ethClient.HeaderByNumber(context.TODO(), nil)
		if err != nil {

		}
		return blockHeader.Number.Uint64(), nil
	case gwcontract.TronGateway:
		latestBlock, err := orc.tronClient.GetLastBlockNumber(context.TODO())
		if err != nil {

		}
		return latestBlock, nil
	}
	return 0, errors.Errorf("invalid gateway type %v", orc.gatewayType)
}

func (orc *Oracle) processEventsByTxHash(solLoomAddr common.Address) error {
	loomcoinDepositEvents, invalidTxHashes, err := orc.getLoomCoinDepositFromTxHash(solLoomAddr)
	if err != nil {
		return err
	}

	if len(invalidTxHashes) > 0 {
		if err := orc.goGateway.ClearInvalidTxHashes(invalidTxHashes); err != nil {
			return err
		}
	}

	if err := orc.goGateway.ProcessDepositByTxHash(loomcoinDepositEvents); err != nil {
		return err
	}

	return nil
}

// Fetches all relevant events from an Ethereum node from startBlock to endBlock (inclusive)
func (orc *Oracle) fetchEvents(startBlock, endBlock uint64) ([]*MainnetEvent, error) {
	// NOTE: Currently either all blocks from w.StartBlock are processed successfully or none are.
	filterOpts := &bind.FilterOpts{
		Start: startBlock,
		End:   &endBlock,
	}

	var erc721Deposits, erc721xDeposits, loomcoinDeposits, erc20Deposits, ethDeposits, withdrawals []*mainnetEventInfo
	var trxDeposits, trc20Deposits []*mainnetEventInfo
	var err error

	switch orc.gatewayType {
	case gwcontract.LoomCoinGateway:
		loomcoinDeposits, err = orc.fetchLoomCoinDeposits(filterOpts)
		if err != nil {
			return nil, err
		}
		withdrawals, err = orc.fetchTokenWithdrawals(filterOpts)
		if err != nil {
			return nil, err
		}

	case gwcontract.EthereumGateway:
		erc721Deposits, err = orc.fetchERC721Deposits(filterOpts)
		if err != nil {
			return nil, err
		}

		erc721xDeposits, err = orc.fetchERC721XDeposits(filterOpts)
		if err != nil {
			return nil, err
		}

		erc20Deposits, err = orc.fetchERC20Deposits(filterOpts)
		if err != nil {
			return nil, err
		}

		ethDeposits, err = orc.fetchETHDeposits(filterOpts)
		if err != nil {
			return nil, err
		}

		withdrawals, err = orc.fetchTokenWithdrawals(filterOpts)
		if err != nil {
			return nil, err
		}

	case gwcontract.TronGateway:
		trxDeposits, err = orc.fetchTRXDeposits(filterOpts)
		if err != nil {
			return nil, err
		}

		trc20Deposits, err = orc.fetchTRC20Deposits(filterOpts)
		if err != nil {
			return nil, err
		}

		withdrawals, err = orc.fetchTronTokenWithdrawals(filterOpts)
		if err != nil {
			return nil, err
		}
	}

	events := make(
		[]*mainnetEventInfo, 0,
		len(erc721Deposits)+len(erc721xDeposits)+len(erc20Deposits)+len(ethDeposits)+len(loomcoinDeposits)+len(withdrawals)+len(trxDeposits)+len(trc20Deposits),
	)

	events = append(erc721Deposits, erc721xDeposits...)
	events = append(events, erc20Deposits...)
	events = append(events, ethDeposits...)
	events = append(events, loomcoinDeposits...)
	events = append(events, withdrawals...)
	events = append(events, trxDeposits...)
	events = append(events, trc20Deposits...)
	sortMainnetEvents(events)

	sortedEvents := make([]*MainnetEvent, len(events))
	for i, event := range events {
		sortedEvents[i] = event.Event
	}

	if len(events) > 0 {
		orc.logger.Debug("fetched Mainnet events",
			"startBlock", startBlock,
			"endBlock", endBlock,
			"erc721-deposits", len(erc721Deposits),
			"erc721x-deposits", len(erc721xDeposits),
			"erc20-deposits", len(erc20Deposits),
			"eth-deposits", len(ethDeposits),
			"loomcoin-deposits", len(loomcoinDeposits),
			"withdrawals", len(withdrawals),
			"trx-deposits", len(trxDeposits),
			"trc20-deposits", len(trc20Deposits),
		)
	}

	return sortedEvents, nil
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
	var err error
	var numEvents int

	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchERC721Deposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "ERC721Received")
	}(time.Now())

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
							TokenID:       &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.TokenId)},
							TxHash:        ev.Raw.TxHash.Bytes(),
						},
					},
				},
			})
		} else {
			err = erc721It.Error()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get event data for ERC721Received")
			}
			erc721It.Close()
			break
		}
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchERC721XDeposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchERC721XDeposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "ERC721XReceived")
	}(time.Now())

	it, err := orc.solGateway.FilterERC721XReceived(filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ERC721XReceived")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := it.Next()
		if ok {
			ev := it.Event
			tokenAddr, err := loom.LocalAddressFromHexString(ev.ContractAddress.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721XReceived token address")
			}
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC721XReceived from address")
			}
			events = append(events, &mainnetEventInfo{
				BlockNum: ev.Raw.BlockNumber,
				TxIdx:    ev.Raw.TxIndex,
				Event: &MainnetEvent{
					EthBlock: ev.Raw.BlockNumber,
					Payload: &MainnetDepositEvent{
						Deposit: &MainnetTokenDeposited{
							TokenKind:     TokenKind_ERC721X,
							TokenContract: loom.Address{ChainID: "eth", Local: tokenAddr}.MarshalPB(),
							TokenOwner:    loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
							TokenID:       &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.TokenId)},
							TokenAmount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
							TxHash:        ev.Raw.TxHash.Bytes(),
						},
					},
				},
			})
		} else {
			err = it.Error()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get event data for ERC721XReceived")
			}
			it.Close()
			break
		}
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchERC20Deposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchERC20Deposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "ERC20Received")
	}(time.Now())

	it, err := orc.solGateway.FilterERC20Received(filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ERC20Received")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := it.Next()
		if ok {
			ev := it.Event
			tokenAddr, err := loom.LocalAddressFromHexString(ev.ContractAddress.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC20Received token address")
			}
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ERC20Received from address")
			}
			events = append(events, &mainnetEventInfo{
				BlockNum: ev.Raw.BlockNumber,
				TxIdx:    ev.Raw.TxIndex,
				Event: &MainnetEvent{
					EthBlock: ev.Raw.BlockNumber,
					Payload: &MainnetDepositEvent{
						Deposit: &MainnetTokenDeposited{
							TokenKind:     TokenKind_ERC20,
							TokenContract: loom.Address{ChainID: "eth", Local: tokenAddr}.MarshalPB(),
							TokenOwner:    loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
							TokenAmount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
							TxHash:        ev.Raw.TxHash.Bytes(),
						},
					},
				},
			})
		} else {
			err = it.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for ERC20Received")
			}
			it.Close()
			break
		}
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchLoomCoinDeposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchLoomCoinDeposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "LoomCoinReceived")
	}(time.Now())

	it, err := orc.solGateway.FilterLoomCoinReceived(filterOpts, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for LoomCoinReceived")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := it.Next()
		if ok {
			ev := it.Event
			tokenAddr, err := loom.LocalAddressFromHexString(ev.LoomCoinAddress.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse LoomCoinReceived token address")
			}
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse LoomCoinReceived from address")
			}
			events = append(events, &mainnetEventInfo{
				BlockNum: ev.Raw.BlockNumber,
				TxIdx:    ev.Raw.TxIndex,
				Event: &MainnetEvent{
					EthBlock: ev.Raw.BlockNumber,
					Payload: &MainnetDepositEvent{
						Deposit: &MainnetTokenDeposited{
							TokenKind:     TokenKind_LoomCoin,
							TokenContract: loom.Address{ChainID: "eth", Local: tokenAddr}.MarshalPB(),
							TokenOwner:    loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
							TokenAmount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
							TxHash:        ev.Raw.TxHash.Bytes(),
						},
					},
				},
			})
		} else {
			err = it.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for LoomCoinReceived")
			}
			it.Close()
			break
		}
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchETHDeposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchETHDeposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "ETHReceived")
	}(time.Now())

	it, err := orc.solGateway.FilterETHReceived(filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for ETHReceived")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := it.Next()
		if ok {
			ev := it.Event
			fromAddr, err := loom.LocalAddressFromHexString(ev.From.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse ETHReceived from address")
			}
			events = append(events, &mainnetEventInfo{
				BlockNum: ev.Raw.BlockNumber,
				TxIdx:    ev.Raw.TxIndex,
				Event: &MainnetEvent{
					EthBlock: ev.Raw.BlockNumber,
					Payload: &MainnetDepositEvent{
						Deposit: &MainnetTokenDeposited{
							TokenKind:   TokenKind_ETH,
							TokenOwner:  loom.Address{ChainID: "eth", Local: fromAddr}.MarshalPB(),
							TokenAmount: &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Amount)},
							TxHash:      ev.Raw.TxHash.Bytes(),
						},
					},
				},
			})
		} else {
			err = it.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for ETHReceived")
			}
			it.Close()
			break
		}
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchTokenWithdrawals(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchTokenWithdrawals", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "TokenWithdrawn")
	}(time.Now())

	it, err := orc.solGateway.FilterTokenWithdrawn(filterOpts, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for TokenWithdrawn")
	}
	events := []*mainnetEventInfo{}
	for {
		ok := it.Next()
		if ok {
			ev := it.Event

			// Not strictly required, but will provide additional protection to oracle in case
			// we get any erc20 events from loomcoin gateway
			if (orc.gatewayType == gwcontract.LoomCoinGateway) && (TokenKind(ev.Kind) != TokenKind_LoomCoin) {
				continue
			}

			tokenAddr, err := loom.LocalAddressFromHexString(ev.ContractAddress.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse TokenWithdrawn token address")
			}
			fromAddr, err := loom.LocalAddressFromHexString(ev.Owner.Hex())
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse TokenWithdrawn from address")
			}

			var tokenID *ltypes.BigUInt
			var amount *ltypes.BigUInt
			switch TokenKind(ev.Kind) {
			case TokenKind_ERC721:
				tokenID = &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Value)}
			// TODO: ERC721X TokenWithdrawn event should probably indicate the token ID... but for
			//       now all we have is the amount.
			case TokenKind_ERC721X, TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin:
				amount = &ltypes.BigUInt{Value: *loom.NewBigUInt(ev.Value)}
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
							TokenID:       tokenID,
							TokenAmount:   amount,
							TxHash:        ev.Raw.TxHash.Bytes(),
						},
					},
				},
			})
		} else {
			err = it.Error()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get event data for TokenWithdrawn")
			}
			it.Close()
			break
		}
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchTRXDeposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchTRXDeposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "TRXReceived")
	}(time.Now())

	contractAddress := orc.cfg.MainnetContractHexAddress
	fromBlock := filterOpts.Start
	toBlock := *filterOpts.End
	filteredEvents, err := orc.tronClient.FilterTRXReceived(contractAddress, fromBlock, toBlock)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get events for TRXReceived")
	}
	events := []*mainnetEventInfo{}
	for _, ev := range filteredEvents {
		fromAddr, err := loom.LocalAddressFromHexString(ev.Result["from"])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse TRXReceived from address")
		}
		amount, ok := loom.NewBigUIntFromInt(0).SetString(ev.Result["amount"], 10)
		if !ok {
			return nil, errors.Wrap(err, "failed to parse TRXReceived amount to big int")
		}
		events = append(events, &mainnetEventInfo{
			BlockNum: uint64(ev.BlockNumber),
			TxIdx:    ev.EventIndex,
			Event: &MainnetEvent{
				EthBlock: uint64(ev.BlockNumber),
				Payload: &MainnetDepositEvent{
					Deposit: &MainnetTokenDeposited{
						TokenKind:     TokenKind_TRX,
						TokenContract: gwcontract.TRXTokenAddr.MarshalPB(),
						TokenOwner:    loom.Address{ChainID: "tron", Local: fromAddr}.MarshalPB(),
						TokenAmount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(amount)},
						TxHash:        []byte(ev.TransactionID),
					},
				},
			},
		})
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchTRC20Deposits(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchTRC20Deposits", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "TRC20Received")
	}(time.Now())

	contractAddress := orc.cfg.MainnetContractHexAddress
	fromBlock := filterOpts.Start
	toBlock := *filterOpts.End
	filteredEvents, err := orc.tronClient.FilterTRC20Received(contractAddress, fromBlock, toBlock)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for TRC20Received")
	}
	events := []*mainnetEventInfo{}
	for _, ev := range filteredEvents {
		fromAddr, err := loom.LocalAddressFromHexString(ev.Result["from"])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse TRC20Received from address")
		}
		tokenAddr, err := loom.LocalAddressFromHexString(ev.Result["contractAddress"])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse TRC20Received from address")
		}
		amount, ok := loom.NewBigUIntFromInt(0).SetString(ev.Result["amount"], 10)
		if !ok {
			return nil, errors.Wrap(err, "failed to parse TRC20Received amount to big int")
		}
		events = append(events, &mainnetEventInfo{
			BlockNum: uint64(ev.BlockNumber),
			TxIdx:    ev.EventIndex,
			Event: &MainnetEvent{
				EthBlock: uint64(ev.BlockNumber),
				Payload: &MainnetDepositEvent{
					Deposit: &MainnetTokenDeposited{
						TokenKind:     TokenKind_TRC20,
						TokenContract: loom.Address{ChainID: "tron", Local: tokenAddr}.MarshalPB(),
						TokenOwner:    loom.Address{ChainID: "tron", Local: fromAddr}.MarshalPB(),
						TokenAmount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(amount)},
						TxHash:        []byte(ev.TransactionID),
					},
				},
			},
		})
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) fetchTronTokenWithdrawals(filterOpts *bind.FilterOpts) ([]*mainnetEventInfo, error) {
	var err error
	var numEvents int
	defer func(begin time.Time) {
		orc.metrics.MethodCalled(begin, "fetchTronTokenWithdrawals", err)
		orc.metrics.FetchedMainnetEvents(numEvents, "TokenWithdrawn")
	}(time.Now())

	contractAddress := orc.cfg.MainnetContractHexAddress
	fromBlock := filterOpts.Start
	toBlock := *filterOpts.End
	filteredEvents, err := orc.tronClient.FilterTokenWithdrawn(contractAddress, fromBlock, toBlock)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs for TokenWithdrawn")
	}

	events := []*mainnetEventInfo{}
	for _, ev := range filteredEvents {
		fromAddr, err := loom.LocalAddressFromHexString(ev.Result["from"])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse TokenWithdrawn from address")
		}
		value, ok := loom.NewBigUIntFromInt(0).SetString(ev.Result["value"], 10)
		if !ok {
			return nil, errors.Wrap(err, "failed to parse TokenWithdrawn value to big int")
		}
		kind, err := strconv.ParseInt(ev.Result["kind"], 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse TokenWithdrawn value to int32")
		}

		var tokenAddr loom.LocalAddress
		if TokenKind(kind) == TokenKind_TRX {
			tokenAddr = gwcontract.TRXTokenAddr.Local
		} else {
			tokenAddr, err = loom.LocalAddressFromHexString(ev.Result["contractAddress"])
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse TokenWithdrawn from address")
			}
		}

		events = append(events, &mainnetEventInfo{
			BlockNum: uint64(ev.BlockNumber),
			TxIdx:    ev.EventIndex,
			Event: &MainnetEvent{
				EthBlock: uint64(ev.BlockNumber),
				Payload: &MainnetWithdrawalEvent{
					Withdrawal: &MainnetTokenWithdrawn{
						TokenKind:     TokenKind(kind),
						TokenContract: loom.Address{ChainID: "tron", Local: tokenAddr}.MarshalPB(),
						TokenOwner:    loom.Address{ChainID: "tron", Local: fromAddr}.MarshalPB(),
						TokenAmount:   &ltypes.BigUInt{Value: *loom.NewBigUInt(value)},
						TxHash:        []byte(ev.TransactionID),
					},
				},
			},
		})
	}
	numEvents = len(events)
	return events, nil
}

func (orc *Oracle) signTransferGatewayWithdrawal(hash []byte) ([]byte, error) {
	var sig []byte
	var err error
	if orc.withdrawalSig == UnprefixedWithdrawalSigType {
		sig, err = lcrypto.SoliditySign(hash, orc.mainnetPrivateKey)
	} else if orc.withdrawalSig == PrefixedWithdrawalSigType {
		sig, err = lcrypto.SoliditySignPrefixed(hash, orc.mainnetPrivateKey)
	} else {
		return nil, errors.New("invalid withdrawal sig type")
	}

	if err != nil {
		return nil, err
	}
	// The first byte should be the signature mode, for details about the signature format refer to
	// https://github.com/loomnetwork/plasma-erc721/blob/master/server/contracts/Libraries/ECVerify.sol
	return append(make([]byte, 1, 66), sig...), nil
}

func LoadDAppChainPrivateKey(hsmEnabled bool, path string) (lcrypto.PrivateKey, error) {
	var privKey lcrypto.PrivateKey
	var err error

	if hsmEnabled {
		privKey, err = lcrypto.LoadYubiHsmPrivKey(path)
	} else {
		privKey, err = lcrypto.LoadEd25519PrivKey(path)
	}

	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func LoadMainnetPrivateKey(hsmEnabled bool, path string) (lcrypto.PrivateKey, error) {
	var privKey lcrypto.PrivateKey
	var err error

	if hsmEnabled {
		privKey, err = lcrypto.LoadYubiHsmPrivKey(path)
	} else {
		privKey, err = lcrypto.LoadSecp256k1PrivKey(path)
	}

	if err != nil {
		return nil, err
	}
	return privKey, nil
}
