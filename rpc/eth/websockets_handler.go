package eth

import (
	"reflect"
	"strings"
)

type WSPRCFunc struct {
	RPCFunc
}

func NewWSRPCFunc(method interface{}, paramNamesString string) *WSPRCFunc {
	var paramNames []string
	if len(paramNamesString) > 0 {
		paramNames = strings.Split(paramNamesString, ",")

	} else {
		paramNames = []string{}
	}

	rMethod := reflect.TypeOf(method)
	if len(paramNames) != rMethod.NumIn() {
		panic("parameter count mismatch making loom api method")
	}
	signature := []reflect.Type{}
	// first parameter is WSRPCCtx
	for p := 1; p < rMethod.NumIn(); p++ {
		signature = append(signature, rMethod.In(p))
	}

	return &WSPRCFunc{
		RPCFunc: RPCFunc{
			method:    reflect.ValueOf(method),
			signature: signature,
		}}
}
