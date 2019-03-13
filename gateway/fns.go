// +build evm

package gateway

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/loomnetwork/loomchain/fnConsensus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client"
	lcrypto "github.com/loomnetwork/go-loom/crypto"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

const MaxWithdrawalToProcess = 128

const SignatureSize = 65

const WithdrawHashSize = 32

func SoliditySignPrefixed(hash []byte, prv lcrypto.PrivateKey) (sig []byte, err error) {
	// Need to prefix the hash with the Ethereum Signed Message
	hash = ssha.SoliditySHA3(
		"\x19Ethereum Signed Message:\n32",
		hash,
	)

	switch prv.(type) {
	case *lcrypto.Secp256k1PrivateKey:
		sig, err = crypto.Sign(hash, prv.(*lcrypto.Secp256k1PrivateKey).ToECDSAPrivKey())
	case *lcrypto.YubiHsmPrivateKey:
		//TODO this feels out of place
		sig, err = lcrypto.YubiHsmSign(hash, prv.(*lcrypto.YubiHsmPrivateKey))
	default:
		return nil, fmt.Errorf("unknown private key type")
	}

	if err != nil {
		return nil, err
	}

	v := sig[len(sig)-1]
	sig[len(sig)-1] = v + 27

	return sig, nil
}

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

func (b *BatchSignWithdrawalFn) decodeCtx(ctx []byte) (int, error) {
	numWithdrawalsToProcess := int(binary.BigEndian.Uint64(ctx))
	if numWithdrawalsToProcess < 0 || numWithdrawalsToProcess > MaxWithdrawalToProcess {
		return 0, fmt.Errorf("invalid ctx")
	}
	return numWithdrawalsToProcess, nil
}

func (b *BatchSignWithdrawalFn) encodeCtx(numPendingWithdrawals int) []byte {
	ctx := make([]byte, 8)
	if numPendingWithdrawals > MaxWithdrawalToProcess {
		numPendingWithdrawals = MaxWithdrawalToProcess
	}
	binary.BigEndian.PutUint64(ctx, uint64(numPendingWithdrawals))
	return ctx
}

func (b *BatchSignWithdrawalFn) PrepareContext() (bool, []byte, error) {
	// Fix number of pending withdrawals we are going to read and sign
	pendingWithdrawals, err := b.goGateway.PendingWithdrawals(b.mainnetGatewayAddress)
	if err != nil {
		return false, nil, err
	}

	b.logger.Error("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! Number of pending withdrawals", "Pending", len(pendingWithdrawals))

	if len(pendingWithdrawals) == 0 {
		return false, nil, nil
	}

	return true, b.encodeCtx(len(pendingWithdrawals)), nil
}

func (b *BatchSignWithdrawalFn) SubmitMultiSignedMessage(ctx []byte, key []byte, signatures [][]byte) {
	numPendingWithdrawalsToProcess, err := b.decodeCtx(ctx)
	if err != nil {
		b.logger.Error("unable to decode ctx")
		return
	}

	message := b.mappedMessage[hex.EncodeToString(key)]
	byteCopied := 0

	tokenOwnersLengthBytes := make([]byte, 8)
	copy(tokenOwnersLengthBytes, message[byteCopied:(byteCopied+len(tokenOwnersLengthBytes))])
	byteCopied += len(tokenOwnersLengthBytes)

	tokenOwnersLength := binary.BigEndian.Uint64(tokenOwnersLengthBytes)

	tokenOwners := make([]byte, int(tokenOwnersLength))
	copy(tokenOwners, message[byteCopied:(byteCopied+len(tokenOwners))])
	byteCopied += len(tokenOwners)

	withdrawalHashesLength := len(message) - byteCopied
	withdrawalHashes := make([]byte, withdrawalHashesLength)
	copy(withdrawalHashes, message[byteCopied:])
	byteCopied += len(withdrawalHashes)

	tokenOwnersArray := strings.Split(string(tokenOwners), "|")

	if len(tokenOwnersArray) != numPendingWithdrawalsToProcess {
		b.logger.Error("internal error, mismatch between tokenOwners array and pending withdrawal to process")
		return
	}

	confirmedWithdrawalRequests := make([]*ConfirmWithdrawalReceiptRequest, len(tokenOwnersArray))

	for i, tokenOwner := range tokenOwnersArray {
		confirmedWithdrawalRequests[i] = &ConfirmWithdrawalReceiptRequest{
			TokenOwner:     loom.MustParseAddress(tokenOwner).MarshalPB(),
			WithdrawalHash: withdrawalHashes[i*WithdrawHashSize : (i+1)*WithdrawHashSize],
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

	b.logger.Error(fmt.Sprintln("@@!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", "Message and signatures:", ctx, key, signatures))

	b.logger.Error("%%!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! Withdrawal Receipt being submitted", "Receipts", fmt.Sprintf("%+v\n", confirmedWithdrawalRequests))

	// TODO: Make contract method to submit all signed withdrawals in batch
	for _, confirmedWithdrawalRequest := range confirmedWithdrawalRequests {
		if err := b.goGateway.ConfirmWithdrawalReceiptV2(confirmedWithdrawalRequest); err != nil {
			b.logger.Error("unable to confirm withdrawal receipt", "error", err)
			break
		}
	}
}

func (b *BatchSignWithdrawalFn) GetMessageAndSignature(ctx []byte) ([]byte, []byte, error) {
	numPendingWithdrawalsToProcess, err := b.decodeCtx(ctx)
	if err != nil {
		return nil, nil, err
	}

	pendingWithdrawals, err := b.goGateway.PendingWithdrawals(b.mainnetGatewayAddress)
	if err != nil {
		return nil, nil, err
	}

	if len(pendingWithdrawals) == 0 {
		return nil, nil, fmt.Errorf("no pending withdrawals, terminating...")
	}

	if len(pendingWithdrawals) < numPendingWithdrawalsToProcess {
		return nil, nil, fmt.Errorf("invalid execution context")
	}

	pendingWithdrawals = pendingWithdrawals[:numPendingWithdrawalsToProcess]

	signature := make([]byte, len(pendingWithdrawals)*SignatureSize)
	withdrawalHashes := make([]byte, len(pendingWithdrawals)*WithdrawHashSize)

	tokenOwnersBuilder := strings.Builder{}

	for i, pendingWithdrawal := range pendingWithdrawals {
		sig, err := SoliditySignPrefixed(pendingWithdrawal.Hash, b.mainnetPrivKey)
		if err != nil {
			return nil, nil, err
		}

		copy(signature[(i*SignatureSize):], sig)
		copy(withdrawalHashes[(i*WithdrawHashSize):], pendingWithdrawal.Hash)

		address := loom.UnmarshalAddressPB(pendingWithdrawal.TokenOwner)
		if i != len(pendingWithdrawals)-1 {
			tokenOwnersBuilder.WriteString(address.String() + "|")
		} else {
			tokenOwnersBuilder.WriteString(address.String())
		}

	}

	tokenOwners := []byte(tokenOwnersBuilder.String())

	tokenOwnersLength := make([]byte, 8)
	binary.BigEndian.PutUint64(tokenOwnersLength, uint64(len(tokenOwners)))

	bytesCopied := 0
	message := make([]byte, len(tokenOwnersLength)+len(tokenOwners)+len(withdrawalHashes))

	copy(message[bytesCopied:], tokenOwnersLength)
	bytesCopied += len(tokenOwnersLength)

	copy(message[bytesCopied:], tokenOwners)
	bytesCopied += len(tokenOwners)

	copy(message[bytesCopied:], withdrawalHashes)
	bytesCopied += len(withdrawalHashes)

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
