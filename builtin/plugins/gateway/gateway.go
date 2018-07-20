// +build evm

package gateway

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/loomnetwork/go-loom/common/evmcompat"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/keystore"
	"github.com/pkg/errors"
)

type (
	InitRequest               = tgtypes.TransferGatewayInitRequest
	GatewayState              = tgtypes.TransferGatewayState
	ProcessEventBatchRequest  = tgtypes.TransferGatewayProcessEventBatchRequest
	GatewayStateRequest       = tgtypes.TransferGatewayStateRequest
	GatewayStateResponse      = tgtypes.TransferGatewayStateResponse
	NFTDeposit                = tgtypes.TransferGatewayNFTDeposit
	TokenDeposit              = tgtypes.TransferGatewayTokenDeposit
	WithdrawERC721Request     = tgtypes.TransferGatewayWithdrawERC721Request
	WithdrawalReceiptRequest  = tgtypes.TransferGatewayWithdrawalReceiptRequest
	WithdrawalReceiptResponse = tgtypes.TransferGatewayWithdrawalReceiptResponse
	WithdrawalReceipt         = tgtypes.TransferGatewayWithdrawalReceipt
	Account                   = tgtypes.TransferGatewayAccount
)

var (
	stateKey = []byte("state")

	errERC20TransferFailed = errors.New("failed to call ERC20 Transfer method")

	// Permissions
	changeOraclesPerm = []byte("change-oracles")
	submitEventsPerm  = []byte("submit-events")

	// Roles
	ownerRole  = "owner"
	oracleRole = "oracle"
)

func accountKey(owner loom.Address) []byte {
	return util.PrefixKey([]byte("account"), owner.Bytes())
}

const erc721ABI = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_tokenId","type":"uint256"}],"name":"getApproved","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_tokenId","type":"uint256"}],"name":"approve","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_tokenId","type":"uint256"}],"name":"transferFrom","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_index","type":"uint256"}],"name":"tokenOfOwnerByIndex","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_tokenId","type":"uint256"}],"name":"safeTransferFrom","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_tokenId","type":"uint256"}],"name":"exists","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_index","type":"uint256"}],"name":"tokenByIndex","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_tokenId","type":"uint256"},{"name":"_data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_from","type":"address"},{"indexed":true,"name":"_to","type":"address"},{"indexed":false,"name":"_tokenId","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_owner","type":"address"},{"indexed":true,"name":"_approved","type":"address"},{"indexed":false,"name":"_tokenId","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_owner","type":"address"},{"indexed":true,"name":"_operator","type":"address"},{"indexed":false,"name":"_approved","type":"bool"}],"name":"ApprovalForAll","type":"event"},{"constant":false,"inputs":[{"name":"_uid","type":"uint256"}],"name":"mint","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("invalid request")
	// ErrPendingWithdrawal indicates that an account already has a withdrawal pending,
	// it must be completed or cancelled before another withdrawal can be started.
	ErrPendingWithdrawal = errors.New("pending withdrawal already exists")
)

// TODO: list of oracles should be editable, the genesis should contain the initial set
type Gateway struct {
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "gateway",
		Version: "0.1.0",
	}, nil
}

func (gw *Gateway) Init(ctx contract.Context, req *InitRequest) error {
	ctx.GrantPermission(changeOraclesPerm, []string{ownerRole})

	for _, oracleAddr := range req.Oracles {
		ctx.GrantPermissionTo(loom.UnmarshalAddressPB(oracleAddr), submitEventsPerm, oracleRole)
	}

	state := &GatewayState{
		LastEthBlock: 0,
	}
	return ctx.Set(stateKey, state)
}

func (gw *Gateway) ProcessEventBatch(ctx contract.Context, req *ProcessEventBatchRequest) error {
	if ok, _ := ctx.HasPermission(submitEventsPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	blockCount := 0           // number of blocks that were actually processed in this batch
	lastEthBlock := uint64(0) // the last block processed in this batch

	for _, deposit := range req.NftDeposits {
		// Events in the batch are expected to be ordered by block, so a batch should contain
		// events from block N, followed by events from block N+1, any other order is invalid.
		if deposit.EthBlock < lastEthBlock {
			return fmt.Errorf("invalid batch, block %v has already been processed", deposit.EthBlock)
		}

		// Multiple validators might submit batches with overlapping block ranges because the
		// Gateway oracles will fetch events from Ethereum at different times, with different
		// latencies, etc. Simply skip blocks that have already been processed.
		if deposit.EthBlock <= state.LastEthBlock {
			continue
		}

		// TODO: figure out if it's a good idea to process the rest of the deposits if one fails
		if err = gw.transferTokenDeposit(ctx, deposit); err != nil {
			ctx.Logger().Error(err.Error())
			continue
		}

		if deposit.EthBlock > lastEthBlock {
			blockCount++
			lastEthBlock = deposit.EthBlock
		}
	}

	// If there are no new events in this batch return an error so that the batch tx isn't
	// propagated to the other nodes.
	if blockCount == 0 {
		return fmt.Errorf("no new events found in the batch")
	}

	state.LastEthBlock = lastEthBlock

	return ctx.Set(stateKey, state)
}

func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}
	return &GatewayStateResponse{State: state}, nil
}

// WithdrawERC721 will attempt to transfer ownership of an ERC721 token to the Gateway contract,
// if it's successful it will store a receipt than can be used to reclaim ownership of the token
// through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another one withdrawal (even if the tokens originate from different
//       ERC721 contracts).
func (gw *Gateway) WithdrawERC721(ctx contract.Context, req *WithdrawERC721Request) error {
	if req.TokenId == nil || req.TokenContract == nil {
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt != nil {
		return ErrPendingWithdrawal
	}

	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerEthAddr, err := resolveToEthAddr(ctx, mapperAddr, ownerAddr)
	if err != nil {
		return err
	}

	tokenEthAddr, err := resolveToEthAddr(ctx, mapperAddr, loom.UnmarshalAddressPB(req.TokenContract))
	if err != nil {
		return err
	}

	// TODO: transfer token back to gateway contract

	ctx.Logger().Info("WithdrawERC721", "owner", ownerEthAddr.Hex(), "token", tokenEthAddr.Hex())

	// generate hash
	hash, err := evmcompat.SoliditySHA3([]*evmcompat.Pair{
		&evmcompat.Pair{Type: "address", Value: ownerEthAddr.Hex()[2:]},
		&evmcompat.Pair{Type: "address", Value: tokenEthAddr.Hex()[2:]},
		&evmcompat.Pair{Type: "uint256", Value: new(big.Int).SetUint64(account.WithdrawalNonce).String()},
		&evmcompat.Pair{Type: "uint256", Value: req.TokenId.Value.Int.String()},
	})
	if err != nil {
		return err
	}

	// Sign the hash with the node's key.
	// The node must be a known validator that's registered with the Mainnet Gateway.
	sig, err := ctx.Sign(hash, keystore.MainnetTransferGatewayKeyID)
	if err != nil {
		return err
	}

	account.WithdrawalReceipt = &WithdrawalReceipt{
		Hash:      hash, // TODO: don't really need to store the hash...
		Signature: sig,
	}
	account.WithdrawalNonce++
	if err := saveAccount(ctx, account); err != nil {
		return err
	}
	return nil
}

// WithdrawalReceipt will return the receipt generated by the last successful call to WithdrawERC721.
// The receipt can be used to reclaim ownership of the token through the Mainnet Gateway.
func (gw *Gateway) WithdrawalReceipt(ctx contract.StaticContext, req *WithdrawalReceiptRequest) (*WithdrawalReceiptResponse, error) {
	// assume the caller is the owner if the request doesn't specify one
	owner := ctx.Message().Sender
	if req.Owner != nil {
		owner = loom.UnmarshalAddressPB(req.Owner)
	}
	if owner.IsEmpty() {
		return nil, errors.New("no owner specified")
	}
	account, err := loadAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &WithdrawalReceiptResponse{Receipt: account.WithdrawalReceipt}, nil
}

func (gw *Gateway) transferTokenDeposit(ctx contract.Context, deposit *NFTDeposit) error {
	// TODO: permissions check

	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	tokenEthAddr := loom.UnmarshalAddressPB(deposit.Token)
	tokenAddr, err := resolveToDAppAddr(ctx, mapperAddr, tokenEthAddr)
	if err != nil {
		return errors.Wrapf(err, "no mapping exists for token %v", tokenEthAddr)
	}

	exists, err := tokenExists(ctx, tokenAddr, deposit.Uid.Value.Int)
	if err != nil {
		return err
	}

	if !exists {
		if err = mintToken(ctx, tokenAddr, deposit.Uid.Value.Int); err != nil {
			return err
		}
	}

	ownerEthAddr := loom.UnmarshalAddressPB(deposit.From)
	ownerAddr, err := resolveToDAppAddr(ctx, mapperAddr, ownerEthAddr)
	if err != nil {
		return errors.Wrapf(err, "no mapping exists for account %v", ownerEthAddr)
	}

	if err = transferToken(ctx, tokenAddr, ownerAddr, deposit.Uid.Value.Int); err != nil {
		return err
	}

	return nil
}

func mintToken(ctx contract.Context, tokenAddr loom.Address, tokenID *big.Int) error {
	_, err := callEVM(ctx, tokenAddr, "mint", tokenID)
	return err
}

func tokenExists(ctx contract.StaticContext, tokenAddr loom.Address, tokenID *big.Int) (bool, error) {
	var result bool
	return result, staticCallEVM(ctx, tokenAddr, "exists", &result, tokenID)
}

func ownerOfToken(ctx contract.StaticContext, tokenAddr loom.Address, tokenID *big.Int) (loom.Address, error) {
	var result common.Address
	if err := staticCallEVM(ctx, tokenAddr, "ownerOf", &result, tokenID); err != nil {
		return loom.Address{}, err
	}
	return loom.Address{
		ChainID: "eth",
		Local:   result.Bytes(),
	}, nil
}

func transferToken(ctx contract.Context, tokenAddr, ownerAddr loom.Address, tokenID *big.Int) error {
	from := common.BytesToAddress(ctx.Message().Sender.Local)
	to := common.BytesToAddress(ownerAddr.Local)
	_, err := callEVM(ctx, tokenAddr, "safeTransferFrom", from, to, tokenID, []byte{})
	return err
}

func callEVM(ctx contract.Context, contractAddr loom.Address, method string, params ...interface{}) ([]byte, error) {
	erc721, err := abi.JSON(strings.NewReader(erc721ABI))
	if err != nil {
		return nil, err
	}
	input, err := erc721.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	var evmOut []byte
	return evmOut, contract.CallEVM(ctx, contractAddr, input, &evmOut)
}

func staticCallEVM(ctx contract.StaticContext, contractAddr loom.Address, method string, result interface{}, params ...interface{}) error {
	erc721, err := abi.JSON(strings.NewReader(erc721ABI))
	if err != nil {
		return err
	}
	input, err := erc721.Pack(method, params...)
	if err != nil {
		return err
	}
	var output []byte
	if err := contract.StaticCallEVM(ctx, contractAddr, input, &output); err != nil {
		return err
	}
	return erc721.Unpack(result, method, output)
}

func loadState(ctx contract.StaticContext) (*GatewayState, error) {
	var state GatewayState
	err := ctx.Get(stateKey, &state)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}
	return &state, nil
}

// Returns the address of the DAppChain account or contract that corresponds to the given Ethereum address
func resolveToDAppAddr(ctx contract.StaticContext, mapperAddr, ethAddr loom.Address) (loom.Address, error) {
	var resp address_mapper.GetMappingResponse
	req := &address_mapper.GetMappingRequest{From: ethAddr.MarshalPB()}
	if err := contract.StaticCallMethod(ctx, mapperAddr, "GetMapping", req, &resp); err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(resp.To), nil
}

// Returns the address of the Ethereum account or contract that corresponds to the given DAppChain address
func resolveToEthAddr(ctx contract.StaticContext, mapperAddr, dappAddr loom.Address) (common.Address, error) {
	var resp address_mapper.GetMappingResponse
	req := &address_mapper.GetMappingRequest{From: dappAddr.MarshalPB()}
	if err := contract.StaticCallMethod(ctx, mapperAddr, "GetMapping", req, &resp); err != nil {
		return common.Address{}, err
	}
	addr := loom.UnmarshalAddressPB(resp.To)
	return common.BytesToAddress(addr.Local), nil
}

func loadAccount(ctx contract.StaticContext, owner loom.Address) (*Account, error) {
	account := Account{Owner: owner.MarshalPB()}
	err := ctx.Get(accountKey(owner), &account)
	if err != nil && err != contract.ErrNotFound {
		return nil, errors.Wrapf(err, "failed to load account for %v", owner)
	}
	return &account, nil
}

func saveAccount(ctx contract.Context, acct *Account) error {
	ownerAddr := loom.UnmarshalAddressPB(acct.Owner)
	if err := ctx.Set(accountKey(ownerAddr), acct); err != nil {
		return errors.Wrapf(err, "failed to save account for %v", ownerAddr)
	}
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
