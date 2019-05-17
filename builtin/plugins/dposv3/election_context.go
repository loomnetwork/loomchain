package dposv3

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
)

type cacheItem struct {
	Value      []byte
	Deleted    bool
	Delegation bool
	P          proto.Message
}

// NOTE: This context doesn't wrap Range(), which means it's not currently possible to use Range()
//       within Elect().
type electionContext struct {
	contract.Context
	pctx      plugin.Context
	cache     map[string]cacheItem
	cachehit  int
	cachemiss int
}

// Preloads DPOS data needed during an election and returns a wrapped contract context that provides
// access to the preloaded data.
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

func (ctx *electionContext) GetAllDelegations() ([]*Delegation, [][]byte, error) {
	delegations := []*Delegation{}
	delegationBytes := [][]byte{}
	log.Error("Starting Range")
	//delegationIdx := make(map[string]*Delegation)
	count := 0
	for _, m := range ctx.Range([]byte("delegation")) {
		var f Delegation
		if bytes.HasSuffix(m.Key, delegationsKey) || len(m.Key) < 3 {
			log.Error(fmt.Sprintf("Skipping delegationsKey -%d", len(m.Key)))
			continue
		}

		if err := proto.Unmarshal(m.Value, &f); err != nil {
			err := errors.Wrapf(err, "unmarshal delegation %s", string(m.Key))
			ctx.Logger().Error(err.Error())
			continue
			//return nil, nil, errors.Wrapf(err, "unmarshal delegation %s", string(m.Key))
		}
		delegations = append(delegations, &f)
		//	delegationBytes = append(delegationBytes, m.Value)
		//Track the index in the array
		//	delegationIdx[fmt.Sprintf("%d-loom.UnmarshalAddressPB(f.GetDelegator()).String()", f.Index)] = &f
		count++
	}
	log.Error(fmt.Sprintf("Ending Range - found %d items", count))

	//TODO making assumption on order based on key order, maybe not
	return delegations, delegationBytes, nil
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

	delegations, _, err := ctx.GetAllDelegations()
	if err != nil {
		return err
	}
	for _, delegation := range delegations {
		delKey, err := computeDelegationsKey(delegation.Index, *delegation.Validator, *delegation.Delegator)
		if err != nil {
			return err
		}
		delKey = append(delegationsKey, delKey...)
		ctx.cache[string(delKey)] = cacheItem{P: delegation, Delegation: true}

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

// Get tries to read the given key from the cache, if the key is not in the cache it will be read
// from the app state. All keys should be in the cache, reading from the app state is just a precaution
// in case electionContext.load() didn't load all the relevant keys.
func (ctx *electionContext) Get(key []byte, pb proto.Message) error {
	if item, exists := ctx.cache[string(key)]; exists {
		if item.Delegation == true {
			ctx.cachehit = ctx.cachehit + 1
			pb = item.P
			return nil
		}
		if len(item.Value) == 0 {
			fmt.Printf("invalid data %s \n", string(key))
			return contract.ErrNotFound
		}
		ctx.cachehit = ctx.cachehit + 1
		return proto.Unmarshal(item.Value, pb)
	}
	ctx.cachemiss = ctx.cachemiss + 1
	return ctx.Context.Get(key, pb)
}

func (ctx *electionContext) Finished() {
	log.Error(fmt.Sprintf("cache hits %d --- cache misses %d", ctx.cachehit, ctx.cachemiss))
}

func (ctx *electionContext) Has(key []byte) bool {
	if item, exists := ctx.cache[string(key)]; exists {
		return !item.Deleted
	}
	return ctx.Context.Has(key)
}

// Set writes the given key to the cache and to the app state. The key is written to the app state
// immediately to ensure the order in which keys are written to the IAVL store during the election
// remains consistent with previous builds that don't have the cache.
func (ctx *electionContext) Set(key []byte, pb proto.Message) error {
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	ctx.cache[string(key)] = cacheItem{Value: data}
	ctx.pctx.Set(key, data)
	return nil
}

func (ctx *electionContext) Delete(key []byte) {
	if key == nil {
		panic("key can't be nil")
	}
	ctx.cache[string(key)] = cacheItem{Deleted: true}
	ctx.pctx.Delete(key)
}
