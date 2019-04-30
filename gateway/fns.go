// +build evm

package gateway

import (
	"encoding/hex"
	"fmt"

	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/loomchain/fnConsensus"

	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	lcrypto "github.com/loomnetwork/go-loom/crypto"
	"github.com/pkg/errors"
)

const MaxWithdrawalToProcess = 20

const SignatureSize = 65

const WithdrawHashSize = 32

type BatchSignWithdrawalFn struct {
	goGateway *DAppChainGateway

	// This could be different for every validator
	mainnetPrivKey lcrypto.PrivateKey

	// Store mapping between key to message
	// This will later used in SubmitMultiSignedMessage
	mappedMessage map[string][]byte

	mainnetGatewayAddress loom.Address

	logger *loom.Logger
}

func (b *BatchSignWithdrawalFn) SubmitMultiSignedMessage(ctx []byte, key []byte, signatures [][]byte) {
	message := b.mappedMessage[hex.EncodeToString(key)]
	if message == nil {
		b.logger.Error("unable to find the message")
		return
	}

	batchWithdrawalFnMessage := &BatchWithdrawalFnMessage{}

	if err := proto.Unmarshal(message, batchWithdrawalFnMessage); err != nil {
		b.logger.Error("unable to unmarshal withdrawal fn message", "error", err)
		return
	}

	confirmedWithdrawalRequests := make([]*ConfirmWithdrawalReceiptRequest, len(batchWithdrawalFnMessage.WithdrawalMessages))

	for i, withdrawalMessage := range batchWithdrawalFnMessage.WithdrawalMessages {
		confirmedWithdrawalRequests[i] = &ConfirmWithdrawalReceiptRequest{
			TokenOwner:     withdrawalMessage.TokenOwner,
			WithdrawalHash: withdrawalMessage.WithdrawalHash,
		}

		validatorSignatures := make([]byte, 0, len(signatures)*SignatureSize)

		for _, signature := range signatures {
			// Validator havent signed
			if signature == nil {
				// Since we are converting to aggregate signature, add zero'ed out bytes for nil signature
				validatorSignatures = append(validatorSignatures, make([]byte, SignatureSize)...)
			} else {
				validatorSignatures = append(validatorSignatures, signature[i*SignatureSize:(i+1)*SignatureSize]...)
			}
		}

		confirmedWithdrawalRequests[i].OracleSignature = validatorSignatures
	}

	b.logger.Info("Withdrawal Receipt being submitted", "Receipts", confirmedWithdrawalRequests)

	// TODO: Make contract method to submit all signed withdrawals in batch
	for _, confirmedWithdrawalRequest := range confirmedWithdrawalRequests {
		if err := b.goGateway.ConfirmWithdrawalReceiptV2(confirmedWithdrawalRequest); err != nil {
			b.logger.Error("unable to confirm withdrawal receipt", "error", err)
			break
		}
	}
}

func (b *BatchSignWithdrawalFn) GetMessageAndSignature(ctx []byte) ([]byte, []byte, error) {
	pendingWithdrawals, err := b.goGateway.PendingWithdrawalsV2(b.mainnetGatewayAddress)
	if err != nil {
		return nil, nil, err
	}

	if len(pendingWithdrawals) == 0 {
		return nil, nil, fmt.Errorf("no pending withdrawals, terminating...")
	}

	numPendingWithdrawalsToProcess := len(pendingWithdrawals)
	if numPendingWithdrawalsToProcess > MaxWithdrawalToProcess {
		numPendingWithdrawalsToProcess = MaxWithdrawalToProcess
	}
	pendingWithdrawals = pendingWithdrawals[:numPendingWithdrawalsToProcess]

	signature := make([]byte, len(pendingWithdrawals)*SignatureSize)

	batchWithdrawalFnMessage := &BatchWithdrawalFnMessage{
		WithdrawalMessages: make([]*WithdrawalMessage, len(pendingWithdrawals)),
	}

	for i, pendingWithdrawal := range pendingWithdrawals {
		sig, err := lcrypto.SoliditySignPrefixed(pendingWithdrawal.Hash, b.mainnetPrivKey)
		if err != nil {
			return nil, nil, err
		}

		copy(signature[(i*SignatureSize):], sig)

		batchWithdrawalFnMessage.WithdrawalMessages[i] = &WithdrawalMessage{}
		batchWithdrawalFnMessage.WithdrawalMessages[i].TokenOwner = pendingWithdrawal.TokenOwner
		batchWithdrawalFnMessage.WithdrawalMessages[i].WithdrawalHash = pendingWithdrawal.Hash
	}

	message, err := proto.Marshal(batchWithdrawalFnMessage)
	if err != nil {
		return nil, nil, err
	}

	return message, signature, nil
}

func (b *BatchSignWithdrawalFn) MapMessage(ctx, key, message []byte) error {
	b.mappedMessage[hex.EncodeToString(key)] = message
	return nil
}

func CreateBatchSignWithdrawalFn(isLoomcoinFn bool, chainID string, fnRegistry fnConsensus.FnRegistry, tgConfig *TransferGatewayConfig, signer auth.Signer) (*BatchSignWithdrawalFn, error) {
	if fnRegistry == nil {
		return nil, fmt.Errorf("unable to start batch sign withdrawal Fn as fn registry is nil")
	}

	if tgConfig == nil || tgConfig.BatchSignFnConfig == nil {
		return nil, fmt.Errorf("unable to start batch sign withdrawal Fn as configuration is invalid")
	}

	fnConfig := tgConfig.BatchSignFnConfig

	mainnetPrivateKey, err := LoadMainnetPrivateKey(fnConfig.MainnetPrivateKeyHsmEnabled, fnConfig.MainnetPrivateKeyPath)
	if err != nil {
		return nil, err
	}

	caller := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(signer.PublicKey()),
	}

	if !common.IsHexAddress(tgConfig.MainnetContractHexAddress) {
		return nil, errors.New("invalid Mainnet Gateway address")
	}

	dappClient := client.NewDAppChainRPCClient(chainID, tgConfig.DAppChainWriteURI, tgConfig.DAppChainReadURI)

	logger := loom.NewLoomLogger(fnConfig.LogLevel, fnConfig.LogDestination)

	var goGateway *DAppChainGateway

	if isLoomcoinFn {
		goGateway, err = ConnectToDAppChainLoomCoinGateway(dappClient, caller, signer, logger)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create dappchain loomcoin gateway")
		}
	} else {
		goGateway, err = ConnectToDAppChainGateway(dappClient, caller, signer, logger)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create dappchain gateway")
		}
	}

	batchWithdrawalFn := &BatchSignWithdrawalFn{
		goGateway:      goGateway,
		mainnetPrivKey: mainnetPrivateKey,
		mappedMessage:  make(map[string][]byte),
		mainnetGatewayAddress: loom.Address{
			ChainID: "eth",
			Local:   common.HexToAddress(tgConfig.MainnetContractHexAddress).Bytes(),
		},
		logger: loom.NewLoomLogger(fnConfig.LogLevel, fnConfig.LogDestination),
	}

	return batchWithdrawalFn, nil
}
