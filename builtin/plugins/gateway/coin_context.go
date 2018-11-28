package gateway

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
)

// Helper for making static calls into the Go contract that stores loom native coins on the DAppChain.
type coinStaticContext struct {
	ctx          contract.StaticContext
	contractAddr loom.Address
}

func newcoinStaticContext(ctx contract.StaticContext) *coinStaticContext {
	contractAddr, err := ctx.Resolve("coin")
	if err != nil {
		panic(err)
	}
	return &coinStaticContext{
		ctx:          ctx,
		contractAddr: contractAddr,
	}
}

func (c *coinStaticContext) balanceOf(owner loom.Address) (*big.Int, error) {
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

// Helper for making calls into the Go contract that stores native loom coins on the DAppChain.
type coinContext struct {
	*coinStaticContext
	ctx contract.Context
}

func newCoinContext(ctx contract.Context) *coinContext {
	return &coinContext{
		coinStaticContext: newcoinStaticContext(ctx),
		ctx:               ctx,
	}
}

func (c *coinContext) transferFrom(from, to loom.Address, amount *big.Int) error {
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

func (c *coinContext) transfer(to loom.Address, amount *big.Int) error {
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

func (c *coinContext) mintToGateway(amount *big.Int) error {
	req := &coin.MintToGatewayRequest{
		Amount: &types.BigUInt{Value: *loom.NewBigUInt(amount)},
	}

	err := contract.CallMethod(c.ctx, c.contractAddr, "MintToGateway", req, nil)
	if err != nil {
		return err
	}

	return nil
}
