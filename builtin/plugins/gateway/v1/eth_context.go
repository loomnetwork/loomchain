package gateway

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/builtin/types/ethcoin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
)

// Helper for making static calls into the Go contract that stores ETH on the DAppChain.
type ethStaticContext struct {
	ctx          contract.StaticContext
	contractAddr loom.Address
}

func newETHStaticContext(ctx contract.StaticContext) *ethStaticContext {
	contractAddr, err := ctx.Resolve("ethcoin")
	if err != nil {
		panic(err)
	}
	return &ethStaticContext{
		ctx:          ctx,
		contractAddr: contractAddr,
	}
}

func (c *ethStaticContext) balanceOf(owner loom.Address) (*big.Int, error) {
	req := &coin.BalanceOfRequest{
		Owner: owner.MarshalPB(),
	}
	var resp coin.BalanceOfResponse
	err := contract.StaticCallMethod(c.ctx, c.contractAddr, "BalanceOf", req, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Balance != nil {
		return resp.Balance.Value.Int, nil
	}
	return nil, nil
}

// Helper for making calls into the Go contract that stores ETH on the DAppChain.
type ethContext struct {
	*ethStaticContext
	ctx contract.Context
}

func newETHContext(ctx contract.Context) *ethContext {
	return &ethContext{
		ethStaticContext: newETHStaticContext(ctx),
		ctx:              ctx,
	}
}

func (c *ethContext) transferFrom(from, to loom.Address, amount *big.Int) error {
	req := &coin.TransferFromRequest{
		From:   from.MarshalPB(),
		To:     to.MarshalPB(),
		Amount: &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	}

	err := contract.CallMethod(c.ctx, c.contractAddr, "TransferFrom", req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *ethContext) transfer(to loom.Address, amount *big.Int) error {
	req := &coin.TransferRequest{
		To:     to.MarshalPB(),
		Amount: &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	}

	err := contract.CallMethod(c.ctx, c.contractAddr, "Transfer", req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *ethContext) mintToGateway(amount *big.Int) error {
	req := &ethcoin.ETHCoinMintToGatewayRequest{
		Amount: &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	}

	err := contract.CallMethod(c.ctx, c.contractAddr, "MintToGateway", req, nil)
	if err != nil {
		return err
	}

	return nil
}
