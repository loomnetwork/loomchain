// +build evm

package auth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom/common/evmcompat"
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
	//fmt.Println("hash ethADdr",hexutil.Encode(hash))

	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	//fmt.Println("verifySolidity66Byte ethADdr",ethAddr.String())

	//signatureNoRecoverID := tx.Signature[1:len(tx.Signature)-1] // remove recovery ID
	//fmt.Println("signature", hexutil.Encode(tx.Signature))
	//fmt.Println("signatureNoRecoverID", hexutil.Encode(signatureNoRecoverID))
	//if !crypto.VerifySignature(tx.PublicKey, hash, signatureNoRecoverID) {
	//	crypto.VerifySignature(tx.PublicKey, hash, signatureNoRecoverID)
	//	return nil, fmt.Errorf("cannot verify transaction signature")
	//}


	return ethAddr.Bytes(), nil
}



