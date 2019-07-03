// +build evm

package auth

import (
	"math/big"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/protobuf/proto"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/types"

	"github.com/loomnetwork/go-loom/vm"

	"github.com/loomnetwork/loomchain/evm/utils"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

const (
	ethID = uint32(4)
)

func VerifySolidity66Byte(tx SignedTx) ([]byte, error) {
	if tx.Signature == nil {
		return verifyEthereumTransacton(tx)
	}
	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifyTron(tx SignedTx) ([]byte, error) {
	tronAddr, err := evmcompat.RecoverAddressFromTypedSig(sha3.SoliditySHA3(tx.Inner), tx.Signature)
	if err != nil {
		return nil, err
	}
	return tronAddr.Bytes(), nil
}

func verifyEthereumTransacton(signedTx SignedTx) ([]byte, error) {
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
	if err := tx.UnmarshalJSON(msg.Data); err != nil {
		return nil, eth.NewErrorf(eth.EcParseError, "Parse params", "unmarshalling ethereum transaction, %v", err)
	}
	chainConfig := utils.DefaultChainConfig()
	ethSigner := etypes.MakeSigner(&chainConfig, big.NewInt(1))
	from, err := etypes.Sender(ethSigner, &tx)
	if err != nil {
		return nil, err
	}
	return from.Bytes(), err
}
