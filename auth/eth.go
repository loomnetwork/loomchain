// +build evm

package auth

import (
	"bytes"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/protobuf/proto"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/types"

	"github.com/loomnetwork/go-loom/vm"

	"github.com/loomnetwork/loomchain/evm/utils"
)

func VerifySolidity66Byte(tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	if tx.Signature == nil {
		return verifyEthTx(tx)
	}
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	tronAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, err
	}
	return tronAddr.Bytes(), nil
}

func verifyBinance(tx SignedTx, allowSigTypes []evmcompat.SignatureType) ([]byte, error) {
	addr, err := evmcompat.RecoverAddressFromTypedSig(evmcompat.GenSHA256(tx.Inner), tx.Signature, allowSigTypes)
	if err != nil {
		return nil, err
	}
	return addr.Bytes(), nil
}

func verifyEthTx(signedTx SignedTx) ([]byte, error) {
	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return nil, err
	}

	var txTx types.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return nil, err
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msg); err != nil {
		return nil, err
	}

	var tx etypes.Transaction
	if err := rlp.DecodeBytes(msg.Data, &tx); err != nil {
		return nil, err
	}

	chainConfig := utils.DefaultChainConfig(true)
	ethSigner := etypes.MakeSigner(&chainConfig, chainConfig.EIP155Block)
	from, err := etypes.Sender(ethSigner, &tx)
	if err != nil {
		return nil, err
	}
	if tx.To() != nil {
		if 0 != bytes.Compare(tx.To().Bytes(), msg.To.Local) {
			return nil, errors.Errorf("to addresses do not match, to.To: %s and msg.To %s", tx.To().String(), msg.To.String())
		}
	}
	if tx.Nonce() != nonceTx.Sequence {
		return nil, errors.Errorf("nonce from tx, %v and nonceTx.Sequence %v", tx.Nonce(), nonceTx.Sequence)
	}

	return from.Bytes(), err
}
