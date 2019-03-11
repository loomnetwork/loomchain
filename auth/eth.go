// +build evm

package auth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

func verifySolidity65Byte(tx SignedTx) ([]byte, error) {
	ethAddr, err := evmcompat.SolidityRecover(tx.PublicKey, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	return ethAddr.Bytes(), nil
}

func verifySolidity66ByteWrong(tx SignedTx) ([]byte, error) {
	hash := crypto.Keccak256(common.BytesToAddress(crypto.Keccak256(tx.PublicKey[1:])[12:]).Bytes())

	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}

func verifySolidity66Byte(tx SignedTx) ([]byte, error) {
	from, to, nonce, err := getFromToNonce(tx)
	if err != nil {
		return nil, errors.Wrapf(err, "retriving hash data from tx %v", tx)
	}
	hash := sha3.SoliditySHA3(
		sha3.Address(common.BytesToAddress(from.Local)),
		sha3.Address(common.BytesToAddress(to.Local)),
		sha3.Uint64(nonce),
		tx.Inner,
	)
	fmt.Println("hash ethADdr",hexutil.Encode(hash))

	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	fmt.Println("verifySolidity66Byte ethADdr",ethAddr.String())
	return ethAddr.Bytes(), nil
}

func getFromToNonce(signedTx SignedTx) (loom.Address, loom.Address, uint64, error) {
	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return loom.Address{}, loom.Address{}, 0, errors.Wrap(err, "unwrap nonce Tx")
	}

	var tx loomchain.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
		return loom.Address{}, loom.Address{}, 0, errors.New("unmarshal tx")
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(tx.Data, &msg); err != nil {
		return loom.Address{}, loom.Address{}, 0, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
	}

	if msg.From == nil {
		return loom.Address{}, loom.Address{}, 0, errors.Errorf("nil from address")
	}

	return loom.UnmarshalAddressPB(msg.From), loom.UnmarshalAddressPB(msg.To), nonceTx.Sequence, nil
}

