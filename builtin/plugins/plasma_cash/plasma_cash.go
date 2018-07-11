// +build evm

package plasma_cash

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	loom "github.com/loomnetwork/go-loom"
	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/mamamerkle"
	"github.com/pkg/errors"
)

type (
	InitRequest                  = pctypes.PlasmaCashInitRequest
	SubmitBlockToMainnetRequest  = pctypes.SubmitBlockToMainnetRequest
	SubmitBlockToMainnetResponse = pctypes.SubmitBlockToMainnetResponse
	GetBlockRequest              = pctypes.GetBlockRequest
	GetBlockResponse             = pctypes.GetBlockResponse
	PlasmaTxRequest              = pctypes.PlasmaTxRequest
	PlasmaTxResponse             = pctypes.PlasmaTxResponse
	DepositRequest               = pctypes.DepositRequest
	PlasmaTx                     = pctypes.PlasmaTx
	GetCurrentBlockResponse      = pctypes.GetCurrentBlockResponse
	GetCurrentBlockRequest       = pctypes.GetCurrentBlockRequest
	PlasmaBookKeeping            = pctypes.PlasmaBookKeeping
	PlasmaBlock                  = pctypes.PlasmaBlock
	Pending                      = pctypes.Pending
	CoinState                    = pctypes.PlasmaCashCoinState
	Coin                         = pctypes.PlasmaCashCoin
	Account                      = pctypes.PlasmaCashAccount
	BalanceOfRequest             = pctypes.PlasmaCashBalanceOfRequest
	BalanceOfResponse            = pctypes.PlasmaCashBalanceOfResponse
	ExitCoinRequest              = pctypes.PlasmaCashExitCoinRequest
	WithdrawCoinRequest          = pctypes.PlasmaCashWithdrawCoinRequest
)

const (
	CoinState_DEPOSITED  = pctypes.PlasmaCashCoinState_DEPOSITED
	CoinState_EXITING    = pctypes.PlasmaCashCoinState_EXITING
	CoinState_CHALLENGED = pctypes.PlasmaCashCoinState_CHALLENGED
	CoinState_EXITED     = pctypes.PlasmaCashCoinState_EXITED
)

type PlasmaCash struct {
}

var (
	blockHeightKey    = []byte("pcash_height")
	pendingTXsKey     = []byte("pcash_pending")
	plasmaMerkleTopic = "pcash_mainnet_merkle"
)

func accountKey(owner loom.Address, contract loom.Address) []byte {
	return util.PrefixKey([]byte("account"), owner.Bytes(), contract.Bytes())
}

func blockKey(height common.BigUInt) []byte {
	return util.PrefixKey([]byte("pcash_block_"), []byte(height.String()))
}

func (c *PlasmaCash) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "plasmacash",
		Version: "1.0.0",
	}, nil
}

func (c *PlasmaCash) Init(ctx contract.Context, req *InitRequest) error {
	//params := req.Params
	ctx.Set(blockHeightKey, &PlasmaBookKeeping{CurrentHeight: &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(0),
	}})

	return nil
}

func round(num, near int64) int64 {
	if num == 0 {
		return near
	}
	if num%near == 0 { //we always want next value
		return num + near
	}
	return ((num + (near - 1)) / near) * near
}

func (c *PlasmaCash) SubmitBlockToMainnet(ctx contract.Context, req *SubmitBlockToMainnetRequest) (*SubmitBlockToMainnetResponse, error) {
	//TODO prevent this being called to oftern

	//if we have a half open block we should flush it
	//Raise blockheight
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	//TODO do this rounding in a bigint safe way
	// round to nearest 1000
	roundedInt := round(pbk.CurrentHeight.Value.Int64(), 1000)
	pbk.CurrentHeight.Value = *loom.NewBigUIntFromInt(roundedInt)
	ctx.Set(blockHeightKey, pbk)

	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)

	leaves := make(map[uint64][]byte)

	for _, v := range pending.Transactions {

		if v.PreviousBlock == nil || v.PreviousBlock.Value.Int64() == int64(0) {
			hash, err := soliditySha3(v.Slot)
			if err != nil {
				return nil, err
			}
			v.MerkleHash = hash
		} else {
			hash, err := rlpEncodeWithSha3(v)
			if err != nil {
				return nil, err
			}
			v.MerkleHash = hash
		}

		leaves[v.Slot] = v.MerkleHash
	}

	smt, err := mamamerkle.NewSparseMerkleTree(64, leaves)
	if err != nil {
		return nil, err
	}

	for _, v := range pending.Transactions {
		v.Proof = smt.CreateMerkleProof(v.Slot)
	}

	merkleHash := smt.Root()

	pb := &PlasmaBlock{
		MerkleHash:   merkleHash,
		Transactions: pending.Transactions,
		Uid:          pbk.CurrentHeight,
	}
	err = ctx.Set(blockKey(pbk.CurrentHeight.Value), pb)
	if err != nil {
		return nil, err
	}

	ctx.EmitTopics(merkleHash, plasmaMerkleTopic)

	//Clear out old pending transactions
	err = ctx.Set(pendingTXsKey, &Pending{})
	if err != nil {
		return nil, err
	}

	return &SubmitBlockToMainnetResponse{MerkleHash: merkleHash}, nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {
	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)
	for _, v := range pending.Transactions {
		if v.Slot == req.Plasmatx.Slot {
			return fmt.Errorf("Error appending plasma transaction with existing slot -%d", v.Slot)
		}
	}
	pending.Transactions = append(pending.Transactions, req.Plasmatx)

	sender := loom.UnmarshalAddressPB(req.Plasmatx.Sender)
	receiver := loom.UnmarshalAddressPB(req.Plasmatx.NewOwner)
	// TODO: get this from the coin history, or require the client to provide it
	contractAddr := loom.RootAddress("eth")

	if err := transferCoin(ctx, req.Plasmatx.Slot, sender, receiver, contractAddr); err != nil {
		return err
	}

	return ctx.Set(pendingTXsKey, pending)
}

func (c *PlasmaCash) DepositRequest(ctx contract.Context, req *DepositRequest) error {
	// TODO: Validate req, must have denomination, from, contract address set

	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)

	// create a new deposit block for the deposit event
	tx := &PlasmaTx{
		Slot:         req.Slot,
		Denomination: req.Denomination,
		NewOwner:     req.From,
		Proof:        make([]byte, 8),
	}

	pb := &PlasmaBlock{
		//MerkleHash:   merkleHash,
		Transactions: []*PlasmaTx{tx},
		Uid:          req.DepositBlock,
	}
	//TODO what if the number scheme is not aligned with our internal!!!!
	//lets add some tests around this
	err := ctx.Set(blockKey(req.DepositBlock.Value), pb)
	if err != nil {
		return err
	}

	// Update the sender's local Plasma account to reflect the deposit
	ownerAddr := loom.UnmarshalAddressPB(req.From)
	contractAddr := loom.UnmarshalAddressPB(req.Contract)
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return errors.Wrap(err, "[PlasmaCash] failed to process deposit")
	}
	account.Coins = append(account.Coins, &Coin{
		Slot:  req.Slot,
		State: CoinState_DEPOSITED,
		Token: req.Denomination,
	})
	if err = saveAccount(ctx, account); err != nil {
		return errors.Wrap(err, "[PlasmaCash] failed to process deposit")
	}

	if req.DepositBlock.Value.Cmp(&pbk.CurrentHeight.Value) > 0 {
		pbk.CurrentHeight.Value = req.DepositBlock.Value
		return ctx.Set(blockHeightKey, pbk)
	}
	return nil
}

// BalanceOf returns the Plasma coins owned by an entity. The request must specifiy the address of
// the token contract for which Plasma coins should be returned.
func (c *PlasmaCash) BalanceOf(ctx contract.StaticContext, req *BalanceOfRequest) (*BalanceOfResponse, error) {
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	contractAddr := loom.UnmarshalAddressPB(req.Contract)
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return nil, errors.Wrap(err, "[PlasmaCash] failed to retrieve coin balance")
	}
	return &BalanceOfResponse{Coins: account.Coins}, nil
}

// ExitCoin updates the state of a Plasma coin from DEPOSITED to EXITING.
// This method should only be called by the Plasma Cash Oracle when it detects an attempted exit
// of a Plasma coin on Ethereum Mainnet.
func (c *PlasmaCash) ExitCoin(ctx contract.Context, req *ExitCoinRequest) error {
	// TODO: Only Oracles should be allowed to call this method.
	defaultErrMsg := "[PlasmaCash] failed to exit coin"
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	contractAddr := loom.UnmarshalAddressPB(req.Contract)
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}
	for _, coin := range account.Coins {
		if coin.Slot == req.Slot {
			if coin.State != CoinState_DEPOSITED {
				return fmt.Errorf("[PlasmaCash] can't exit coin %v in state %s", coin.Slot, coin.State)
			}

			coin.State = CoinState_EXITING

			if err = saveAccount(ctx, account); err != nil {
				return errors.Wrap(err, defaultErrMsg)
			}
			return nil
		}
	}
	return errors.New(defaultErrMsg)
}

// WithdrawCoin removes a Plasma coin from a local Plasma account.
// This method should only be called by the Plasma Cash Oracle when it detects a withdrawal of a
// Plasma coin on Ethereum Mainnet.
func (c *PlasmaCash) WithdrawCoin(ctx contract.Context, req *WithdrawCoinRequest) error {
	// TODO: Only Oracles should be allowed to call this method.
	defaultErrMsg := "[PlasmaCash] failed to withdraw coin"
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	contractAddr := loom.UnmarshalAddressPB(req.Contract)
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}
	for i, coin := range account.Coins {
		if coin.Slot == req.Slot {
			// NOTE: We don't require the coin to be in EXITED state to process the withdrawal
			// because the owner is free (in theory) to initiate an exit without involving the
			// DAppChain.
			account.Coins[i] = account.Coins[len(account.Coins)-1]
			account.Coins = account.Coins[:len(account.Coins)-1]

			if err = saveAccount(ctx, account); err != nil {
				return errors.Wrap(err, defaultErrMsg)
			}
			return nil
		}
	}
	return errors.New(defaultErrMsg)
}

func (c *PlasmaCash) GetCurrentBlockRequest(ctx contract.StaticContext, req *GetCurrentBlockRequest) (*GetCurrentBlockResponse, error) {
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)
	return &GetCurrentBlockResponse{pbk.CurrentHeight}, nil
}

func (c *PlasmaCash) GetBlockRequest(ctx contract.StaticContext, req *GetBlockRequest) (*GetBlockResponse, error) {
	pb := &PlasmaBlock{}

	err := ctx.Get(blockKey(req.BlockHeight.Value), pb)
	if err != nil {
		return nil, err
	}

	return &GetBlockResponse{Block: pb}, nil
}

func loadAccount(ctx contract.StaticContext, ownerAddr, contractAddr loom.Address) (*Account, error) {
	account := &Account{
		Owner:    ownerAddr.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
	}
	err := ctx.Get(accountKey(ownerAddr, contractAddr), account)
	if err != nil && err != contract.ErrNotFound {
		return nil, errors.Wrapf(err, "failed to load account for %s, %s",
			ownerAddr, contractAddr)
	}
	return account, nil
}

func saveAccount(ctx contract.Context, acct *Account) error {
	ownerAddr := loom.UnmarshalAddressPB(acct.Owner)
	contractAddr := loom.UnmarshalAddressPB(acct.Contract)
	if err := ctx.Set(accountKey(ownerAddr, contractAddr), acct); err != nil {
		return errors.Wrapf(err, "failed to save account for %s, %s",
			ownerAddr, contractAddr)
	}
	return nil
}

// Updates the sender's and receiver's local Plasma accounts to reflect a Plasma coin transfer.
func transferCoin(ctx contract.Context, slot uint64, sender, receiver, contractAddr loom.Address) error {
	fromAcct, err := loadAccount(ctx, sender, contractAddr)
	if err != nil {
		return err
	}

	coinIdx := -1
	for i, coin := range fromAcct.Coins {
		if coin.Slot == slot {
			if coin.State != CoinState_DEPOSITED {
				return fmt.Errorf("can't transfer coin %v in state %s", slot, coin.State)
			}
			coinIdx = i
			break
		}
	}
	if coinIdx == -1 {
		return fmt.Errorf("can't transfer coin %v: sender doesn't own it", slot)
	}

	toAcct, err := loadAccount(ctx, receiver, contractAddr)
	if err != nil {
		return err
	}

	fromCoins := fromAcct.Coins
	toAcct.Coins = append(toAcct.Coins, fromCoins[coinIdx])
	fromCoins[coinIdx] = fromCoins[len(fromCoins)-1]
	fromAcct.Coins = fromCoins[:len(fromCoins)-1]

	if err := saveAccount(ctx, fromAcct); err != nil {
		return errors.Wrap(err, "failed to transfer coin %v: can't save sender account")
	}
	if err := saveAccount(ctx, toAcct); err != nil {
		return errors.Wrap(err, "failed to transfer coin %v: can't save receiver account")
	}

	return nil
}

func soliditySha3(data uint64) ([]byte, error) {
	pairs := []*evmcompat.Pair{&evmcompat.Pair{"uint64", strconv.FormatUint(data, 10)}}
	hash, err := evmcompat.SoliditySHA3(pairs)
	if err != nil {
		return []byte{}, err
	}
	return hash, err
}

func rlpEncodeWithSha3(pb *PlasmaTx) ([]byte, error) {
	hash, err := rlpEncode(pb)
	if err != nil {
		return []byte{}, err
	}
	d := sha3.NewKeccak256()
	d.Write(hash)
	return d.Sum(nil), nil
}

func rlpEncode(pb *PlasmaTx) ([]byte, error) {
	return rlp.EncodeToBytes([]interface{}{
		uint64(pb.Slot),
		pb.PreviousBlock.Value.Bytes(),
		uint32(pb.Denomination.Value.Int64()),
		pb.GetNewOwner().Local,
	})
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
