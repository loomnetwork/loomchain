// +build evm

package plasma_cash

import (
	"bytes"
	"encoding/binary"
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

func coinKey(slot uint64) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, slot)
	return util.PrefixKey([]byte("coin"), buf.Bytes())
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
	defaultErrMsg := "[PlasmaCash] failed to process transfer"
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
	coin, err := loadCoin(ctx, req.Plasmatx.Slot)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}

	ctx.Logger().Debug(fmt.Sprintf("Transfer %v from %v to %v", coin.Slot, sender, receiver))
	if err := transferCoin(ctx, coin, sender, receiver); err != nil {
		return errors.Wrap(err, defaultErrMsg)
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

	defaultErrMsg := "[PlasmaCash] failed to process deposit"
	// Update the sender's local Plasma account to reflect the deposit
	ownerAddr := loom.UnmarshalAddressPB(req.From)
	contractAddr := loom.UnmarshalAddressPB(req.Contract)
	ctx.Logger().Debug(fmt.Sprintf("Deposit %v from %v", req.Slot, ownerAddr))
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}
	err = saveCoin(ctx, &Coin{
		Slot:     req.Slot,
		State:    CoinState_DEPOSITED,
		Token:    req.Denomination,
		Contract: req.Contract,
	})
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}
	account.Slots = append(account.Slots, req.Slot)
	if err = saveAccount(ctx, account); err != nil {
		return errors.Wrap(err, defaultErrMsg)
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
	coins := make([]*Coin, 0, len(account.Slots))
	for _, slot := range account.Slots {
		coin, err := loadCoin(ctx, slot)
		if err != nil {
			ctx.Logger().Error(err.Error())
		}
		coins = append(coins, coin)
	}
	return &BalanceOfResponse{Coins: coins}, nil
}

// ExitCoin updates the state of a Plasma coin from DEPOSITED to EXITING.
// This method should only be called by the Plasma Cash Oracle when it detects an attempted exit
// of a Plasma coin on Ethereum Mainnet.
func (c *PlasmaCash) ExitCoin(ctx contract.Context, req *ExitCoinRequest) error {
	// TODO: Only Oracles should be allowed to call this method.
	defaultErrMsg := "[PlasmaCash] failed to exit coin"

	coin, err := loadCoin(ctx, req.Slot)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}

	if coin.State != CoinState_DEPOSITED {
		return fmt.Errorf("[PlasmaCash] can't exit coin %v in state %s", coin.Slot, coin.State)
	}

	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	contractAddr := loom.UnmarshalAddressPB(coin.Contract)
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}

	for _, slot := range account.Slots {
		if slot == coin.Slot {
			coin.State = CoinState_EXITING

			if err = saveCoin(ctx, coin); err != nil {
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

	coin, err := loadCoin(ctx, req.Slot)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	contractAddr := loom.UnmarshalAddressPB(coin.Contract)
	account, err := loadAccount(ctx, ownerAddr, contractAddr)
	if err != nil {
		return errors.Wrap(err, defaultErrMsg)
	}
	for i, slot := range account.Slots {
		if slot == coin.Slot {
			// NOTE: We don't require the coin to be in EXITED state to process the withdrawal
			// because the owner is free (in theory) to initiate an exit without involving the
			// DAppChain.
			account.Slots[i] = account.Slots[len(account.Slots)-1]
			account.Slots = account.Slots[:len(account.Slots)-1]

			if err = saveAccount(ctx, account); err != nil {
				return errors.Wrap(err, defaultErrMsg)
			}
			ctx.Delete(coinKey(slot))
			return nil
		}
	}
	return errors.New(defaultErrMsg)
}

func (c *PlasmaCash) GetCurrentBlockRequest(ctx contract.StaticContext, req *GetCurrentBlockRequest) (*GetCurrentBlockResponse, error) {
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)
	return &GetCurrentBlockResponse{BlockHeight: pbk.CurrentHeight}, nil
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

func loadCoin(ctx contract.StaticContext, slot uint64) (*Coin, error) {
	coin := &Coin{}
	err := ctx.Get(coinKey(slot), coin)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load coin %v", coin.Slot)
	}
	return coin, nil
}

func saveCoin(ctx contract.Context, coin *Coin) error {
	if err := ctx.Set(coinKey(coin.Slot), coin); err != nil {
		return errors.Wrapf(err, "failed to save coin %v", coin.Slot)
	}
	return nil
}

// Updates the sender's and receiver's local Plasma accounts to reflect a Plasma coin transfer.
func transferCoin(ctx contract.Context, coin *Coin, sender, receiver loom.Address) error {
	if coin.State != CoinState_DEPOSITED {
		return fmt.Errorf("can't transfer coin %v in state %s", coin.Slot, coin.State)
	}

	contractAddr := loom.UnmarshalAddressPB(coin.Contract)
	fromAcct, err := loadAccount(ctx, sender, contractAddr)
	if err != nil {
		return err
	}

	coinIdx := -1
	for i, slot := range fromAcct.Slots {
		if slot == coin.Slot {
			coinIdx = i
			break
		}
	}
	if coinIdx == -1 {
		return fmt.Errorf("can't transfer coin %v: sender doesn't own it", coin.Slot)
	}

	toAcct, err := loadAccount(ctx, receiver, contractAddr)
	if err != nil {
		return err
	}

	fromSlots := fromAcct.Slots
	toAcct.Slots = append(toAcct.Slots, fromSlots[coinIdx])
	fromSlots[coinIdx] = fromSlots[len(fromSlots)-1]
	fromAcct.Slots = fromSlots[:len(fromSlots)-1]

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
