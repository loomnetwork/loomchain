// +build evm

package auth

import (
	"bytes"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/protobuf/proto"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

func VerifySolidity66Byte(chainID string, tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	if tx.Signature == nil {
		return verifyEthTx(chainID, tx)
	}
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(chainID string, tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	tronAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, err
	}
	return tronAddr.Bytes(), nil
}

func verifyBinance(chainID string, tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	addr, err := evmcompat.RecoverAddressFromTypedSig(evmcompat.GenSHA256(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, err
	}
	return addr.Bytes(), nil
}

func verifyEthTx(chainID string, signedTx SignedTx) ([]byte, error) {
	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return nil, err
	}

	var txTx types.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return nil, err
	}

	var msgTx vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msgTx); err != nil {
		return nil, err
	}

	var ethTx etypes.Transaction
	if err := rlp.DecodeBytes(msgTx.Data, &ethTx); err != nil {
		return nil, errors.Wrap(err, "failed to decode EthereumTx")
	}

	if ethTx.To() != nil && !bytes.Equal(ethTx.To().Bytes(), msgTx.To.Local) {
		return nil, errors.Errorf(
			"EthereumTx.To (%s) doesn't match MessageTx.To (%s)",
			ethTx.To().String(), msgTx.To.String(),
		)
	}
	if ethTx.Nonce() != nonceTx.Sequence {
		return nil, errors.Errorf(
			"EthereumTx.Nonce (%d) doesn't match NonceTx.Sequence (%d)",
			ethTx.Nonce(), nonceTx.Sequence,
		)
	}

	ethChainID, err := evmcompat.ToEthereumChainID(chainID)
	if err != nil {
		return nil, err
	}
	ethSigner := etypes.NewEIP155Signer(ethChainID)
	from, err := etypes.Sender(ethSigner, &ethTx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to recover signer from EthereumTx")
	}

	return from.Bytes(), err
}
