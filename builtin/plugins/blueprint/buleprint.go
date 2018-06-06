package blueprint

import (
	"encoding/json"
	"strings"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/blueprint"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
)

type BluePrint struct {
}

func (e *BluePrint) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "BluePrint",
		Version: "1.0.0",
	}, nil
}

func (e *BluePrint) Init(ctx contract.Context, req *plugin.Request) error {
	return nil
}

func (e *BluePrint) CreateAccount(ctx contract.Context, accTx *types.BluePrintCreateAccountTx) error {
	owner := strings.TrimSpace(accTx.Owner)
	// confirm owner doesnt exist already
	if ctx.Has(e.ownerKey(owner)) {
		return errors.New("Owner already exists")
	}
	addr := []byte(ctx.Message().Sender.Local)
	state := &types.BluePrintAppState{
		Address: addr,
	}
	if err := ctx.Set(e.ownerKey(owner), state); err != nil {
		return errors.Wrap(err, "Error setting state")
	}
	emitMsg := struct {
		Owner  string
		Method string
		Addr   []byte
	}{owner, "createacct", addr}
	emitMsgJSON, err := json.Marshal(&emitMsg)

	if err != nil {
		log.Default.Error("Error marshalling emit message")
	}
	ctx.Emit(emitMsgJSON)
	return nil
}

func (e *BluePrint) SaveState(ctx contract.Context, tx *types.BluePrintStateTx) error {
	owner := strings.TrimSpace(tx.Owner)
	var curState types.BluePrintAppState
	if err := ctx.Get(e.ownerKey(owner), &curState); err != nil {
		return err
	}
	if loom.LocalAddress(curState.Address).Compare(ctx.Message().Sender.Local) != 0 {
		return errors.New("Owner unverified")
	}
	curState.Blob = tx.Data
	if err := ctx.Set(e.ownerKey(owner), &curState); err != nil {
		return errors.Wrap(err, "Error marshaling state node")
	}
	emitMsg := struct {
		Owner  string
		Method string
		Addr   []byte
		Value  int64
	}{Owner: owner, Method: "savestate", Addr: curState.Address}
	if err := json.Unmarshal(tx.Data, &emitMsg); err != nil {
		return err
	}
	emitMsgJSON, err := json.Marshal(&emitMsg)
	if err != nil {
		log.Default.Error("Error marshalling emit message")
	}
	ctx.Emit(emitMsgJSON)

	return nil
}

func (e *BluePrint) GetState(ctx contract.StaticContext, params *types.StateQueryParams) (*types.StateQueryResult, error) {
	if ctx.Has(e.ownerKey(params.Owner)) {
		var curState types.BluePrintAppState
		if err := ctx.Get(e.ownerKey(params.Owner), &curState); err != nil {
			return nil, err
		}
		return &types.StateQueryResult{State: curState.Blob}, nil
	}
	return &types.StateQueryResult{}, nil
}

func (s *BluePrint) ownerKey(owner string) []byte {
	return []byte("owner:" + owner)
}

func (e *BluePrint) SetMsg(ctx contract.Context, req *types.MapEntry) error {
	eventData := struct {
		Method string
		Key    string
		Value  string
	}{Method: "SetMsg", Key: req.Key, Value: req.Value}
	eventJSON, err := json.Marshal(&eventData)
	if err != nil {
		log.Default.Error("Error marshalling emit message")
	}
	ctx.Emit(eventJSON)

	return ctx.Set([]byte(req.Key), req)
}

func (e *BluePrint) SetMsgEcho(ctx contract.Context, req *types.MapEntry) (*types.MapEntry, error) {
	eventData := struct {
		Method string
		Key    string
		Value  string
	}{Method: "SetMsgEcho", Key: req.Key, Value: req.Value}
	eventJSON, err := json.Marshal(&eventData)
	if err != nil {
		log.Default.Error("Error marshalling emit message")
	}
	ctx.Emit(eventJSON)

	if err := ctx.Set([]byte(req.Key), req); err != nil {
		return nil, err
	}
	return &types.MapEntry{
		Key:   req.Key,
		Value: req.Value,
	}, nil
}

func (e *BluePrint) GetMsg(ctx contract.StaticContext, req *types.MapEntry) (*types.MapEntry, error) {
	var result types.MapEntry
	if err := ctx.Get([]byte(req.Key), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&BluePrint{})
