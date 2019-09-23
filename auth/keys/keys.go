package keys

import (
	"context"

	"github.com/loomnetwork/go-loom"
)

type SignedTxType string

const (
	LoomSignedTxType     SignedTxType = "loom"
	EthereumSignedTxType SignedTxType = "eth"
	TronSignedTxType     SignedTxType = "tron"
	BinanceSignedTxType  SignedTxType = "binance"
)

// AccountType is used to specify which address should be used on-chain to identify a tx sender.
type AccountType int

const (
	// NativeAccountType indicates that the tx sender address should be passed through to contracts
	// as is, with the original chain ID intact.
	NativeAccountType AccountType = iota
	// MappedAccountType indicates that the tx sender address should be mapped to a DAppChain
	// address before being passed through to contracts.
	MappedAccountType
)

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	ContextKeyOrigin  = contextKey("origin")
	ContextKeyCheckTx = contextKey("CheckTx")
)

func Origin(ctx context.Context) loom.Address {
	return ctx.Value(ContextKeyOrigin).(loom.Address)
}
