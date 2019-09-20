package loomchain

import (
	"sort"

	appstate "github.com/loomnetwork/loomchain/state"
)

type forkRoute struct {
	TxHandler
	Height int64
}

type forkList []forkRoute

func (s forkList) Len() int {
	return len(s)
}

func (s forkList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s forkList) Less(i, j int) bool {
	return s[i].Height < s[j].Height
}

type ForkRouter struct {
	routes map[string]forkList
}

func NewForkRouter() *ForkRouter {
	return &ForkRouter{
		routes: make(map[string]forkList),
	}
}

func (r *ForkRouter) Handle(chainID string, height int64, handler TxHandler) {
	routes := r.routes[chainID]
	found := sort.Search(len(routes), func(i int) bool {
		return routes[i].Height >= height
	})
	if found < len(routes) && routes[found].Height == height {
		panic("route already exists for given chain and height")
	}
	routes = append(routes, forkRoute{
		TxHandler: handler,
		Height:    height,
	})
	sort.Sort(routes)

	r.routes[chainID] = routes
}

func (r *ForkRouter) ProcessTx(state appstate.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	block := state.Block()
	routes := r.routes[block.ChainID]

	var found TxHandler
	for _, route := range routes {
		if route.Height > block.Height {
			break
		}
		found = route
	}

	return found.ProcessTx(state, txBytes, isCheckTx)
}
