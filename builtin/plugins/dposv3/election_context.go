package dposv3

import (
	"reflect"
	"sort"

	"github.com/gogo/protobuf/proto"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

/*
type actionKind int

const (
	setAction actionKind = iota
	delAction
)

type action struct {
	kind       actionKind
	key, value []byte
}
*/

type cacheItem struct {
	Value   []byte
	Deleted bool
}

// NOTE: This context doesn't wrap Range(), which means it's not currently possible to use Range()
//       within Elect().
type electionContext struct {
	contract.Context
	pctx  plugin.Context
	cache map[string]cacheItem
	// Was going to retain order of actions so this context would be backwards compatible, but
	// it's not possible because the election will also call the coin contract, which won't use this
	// context, so the order in which writes occur will change anyway.
	// actions []action
}

func newElectionContext(ctx contract.Context) (*electionContext, error) {
	// Need the plugin.Context to skip protobuf de/serialization when loading & storing keys
	pctx := reflect.Indirect(reflect.ValueOf(ctx)).FieldByName("Context").Interface().(plugin.Context)
	ectx := &electionContext{
		Context: ctx,
		pctx:    pctx,
		cache:   make(map[string]cacheItem),
	}
	return ectx, ectx.load()
}

func (ctx *electionContext) load() error {
	pctx := ctx.pctx

	data := pctx.Get(stateKey)
	if len(data) > 0 {
		ctx.cache[string(stateKey)] = cacheItem{Value: data}
	}

	var delIdxList dtypes.DelegationList
	data = pctx.Get(delegationsKey)
	if len(data) > 0 {
		ctx.cache[string(delegationsKey)] = cacheItem{Value: data}
		if err := proto.Unmarshal(data, &delIdxList); err != nil {
			return err
		}
	}

	for _, del := range delIdxList.Delegations {
		var delegation Delegation
		delKey, err := computeDelegationsKey(del.Index, *del.Validator, *del.Delegator)
		if err != nil {
			return err
		}
		delKey = append(delegationsKey, delKey...)
		data := pctx.Get(delKey)
		if len(data) > 0 {
			ctx.cache[string(delKey)] = cacheItem{Value: data}
			if err := proto.Unmarshal(data, &delegation); err != nil {
				return err
			}
		}

		if len(delegation.Referrer) > 0 {
			refKey := append(referrersKey, delegation.Referrer...)
			if _, seen := ctx.cache[string(refKey)]; !seen {
				data := pctx.Get(refKey)
				if len(data) > 0 {
					ctx.cache[string(refKey)] = cacheItem{Value: data}
				}
			}
		}
	}

	var candidateList dtypes.CandidateList
	data = pctx.Get(candidatesKey)
	if len(data) > 0 {
		ctx.cache[string(candidatesKey)] = cacheItem{Value: data}
		if err := proto.Unmarshal(data, &candidateList); err != nil {
			return err
		}
	}

	for _, cand := range candidateList.Candidates {
		validatorStatsKey := append(statisticsKey, cand.Address.Local...)
		data := pctx.Get(validatorStatsKey)
		if len(data) > 0 {
			ctx.cache[string(validatorStatsKey)] = cacheItem{Value: data}
		}
	}

	return nil
}

func (ctx *electionContext) Flush() error {
	keys := make([]string, len(ctx.cache))
	i := 0
	for k := range ctx.cache {
		keys[i] = k
		i++
	}

	sort.Strings(keys)
	for _, k := range keys {
		if item, exists := ctx.cache[k]; exists {
			if item.Deleted {
				ctx.pctx.Delete([]byte(k))
			} else {
				ctx.pctx.Set([]byte(k), item.Value)
			}
		}
	}
	return nil
}

func (ctx *electionContext) Get(key []byte, pb proto.Message) error {
	if item, exists := ctx.cache[string(key)]; exists {
		if len(item.Value) == 0 {
			return contract.ErrNotFound
		}
		return proto.Unmarshal(item.Value, pb)
	}
	return contract.ErrNotFound
}

func (ctx *electionContext) Has(key []byte) bool {
	if item, exists := ctx.cache[string(key)]; exists {
		return !item.Deleted
	}
	return false
}

func (ctx *electionContext) Set(key []byte, pb proto.Message) error {
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	ctx.cache[string(key)] = cacheItem{Value: data}
	return nil
}

func (ctx *electionContext) Delete(key []byte) {
	if key == nil {
		panic("key can't be nil")
	}
	ctx.cache[string(key)] = cacheItem{Deleted: true}
}
