package plugin

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/perlin-network/life/gowasm"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
)

type WASMLoader struct {
	Dir       string
	contracts map[string]*WASMContractClient
	sync.Mutex
}

func NewWASMLoader(dir string) *WASMLoader {
	return &WASMLoader{
		Dir:       dir,
		contracts: make(map[string]*WASMContractClient),
	}
}

func (loader *WASMLoader) LoadContract(name string) (plugin.Contract, error) {
	loader.Lock()
	defer loader.Unlock()

	contract, ok := loader.contracts[name]
	if ok {
		return contract, nil
	}

	contract, err := loader.loadContractFull(name)
	if err != nil {
		return nil, err
	}
	loader.contracts[name] = contract
	return contract, nil
}

func (loader *WASMLoader) UnloadContracts() {
}

func (loader *WASMLoader) loadContractFull(name string) (*WASMContractClient, error) {
	path, err := discoverWASM(loader.Dir, name)
	if err != nil {
		return nil, ErrPluginNotFound
	}

	return &WASMContractClient{
		path: path,
	}, nil
}

func isWASMMatch(f os.FileInfo, meta *plugin.Meta) bool {
	if f.IsDir() && f.Size() <= 0 {
		return false
	}

	info, err := parseFileName(f.Name())
	if err != nil {
		return false
	}

	if info.Ext != ".wasm" {
		return false
	}

	if info.Base == meta.Name && info.Version == meta.Version {
		return true
	}

	return false
}

func discoverWASM(dir string, name string) (string, error) {
	meta, err := ParseMeta(name)
	if err != nil {
		return "", err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if isWASMMatch(file, meta) {
			return path.Join(dir, file.Name()), nil
		}
	}

	return "", ErrPluginNotFound
}

var _ Loader = &WASMLoader{}

type WASMContractClient struct {
	path string
}

func (c *WASMContractClient) apiRequestFn(sctx plugin.StaticContext, ctx plugin.Context, errOut *error) func(args ...gowasm.Value) interface{} {
	apiServer := &GRPCAPIServer{
		sctx: sctx,
		ctx:  ctx,
	}

	asv := reflect.ValueOf(apiServer)
	return func(args ...gowasm.Value) interface{} {
		if len(args) < 1 {
			return gowasm.NewJSError(fmt.Errorf("api request: invalid num of args"))
		}
		data := args[0].Bytes()
		reader := bytes.NewBuffer(data)
		var methodLen, argc int16
		err := binary.Read(reader, binary.BigEndian, &methodLen)
		if err != nil {
			*errOut = err
			return gowasm.NewJSError(err)
		}
		methodBytes := make([]byte, methodLen)
		_, err = reader.Read(methodBytes)
		if err != nil {
			*errOut = err
			return gowasm.NewJSError(err)
		}

		methodByName := asv.MethodByName(string(methodBytes))
		if !methodByName.IsValid() {
			*errOut = fmt.Errorf("api request: method %s not exists", methodBytes)
			return gowasm.NewJSError(*errOut)
		}

		err = binary.Read(reader, binary.BigEndian, &argc)
		if err != nil {
			*errOut = err
			return gowasm.NewJSError(err)
		}
		arguments := make([]reflect.Value, int(argc+1))
		arguments[0] = reflect.ValueOf(context.TODO())
		if argc > 0 {
			for i := 0; i < int(argc); i++ {
				var l int16
				err = binary.Read(reader, binary.BigEndian, &l)
				if err != nil {
					*errOut = err
					return gowasm.NewJSError(err)
				}
				d := make([]byte, l)
				err = binary.Read(reader, binary.BigEndian, &d)
				if err != nil {
					*errOut = err
					return gowasm.NewJSError(err)
				}

				arg := methodByName.Type().In(1 + i).Elem()
				r := reflect.New(arg)

				message := r.Interface().(proto.Message)
				err := proto.Unmarshal(d, message)
				if err != nil {
					*errOut = err
					return gowasm.NewJSError(err)
				}
				arguments[1+i] = r
			}
		}
		out := methodByName.Call(arguments)
		if len(out) != 2 {
			*errOut = errors.New("api request: invalid len of out")
			return gowasm.NewJSError(*errOut)
		}
		if !out[1].IsNil() {
			*errOut = fmt.Errorf("api request: call %s return err %v", methodBytes, out[1].Interface())
			return gowasm.NewJSError(*errOut)
		}
		outData, err := proto.Marshal(out[0].Interface().(proto.Message))
		if err != nil {
			*errOut = err
			return gowasm.NewJSError(err)
		}
		return gowasm.ByteSlice2JSArray(outData)
	}
}

func (c *WASMContractClient) newContractCallHandler(method string, sctx plugin.StaticContext, ctx plugin.Context, req *plugin.Request, resp **plugin.Response, err *error) gowasm.JSObject {
	return gowasm.JSObject{
		"GetMethod": func(args ...gowasm.Value) interface{} {
			return method
		},
		"GetRequest": func(args ...gowasm.Value) interface{} {
			if sctx == nil || req == nil {
				return nil
			}
			request := makeContext(sctx, req, 1)
			bytes, e := proto.Marshal(request)
			if e != nil {
				*err = e
				return gowasm.NewJSError(e)
			}
			is := make([]interface{}, len(bytes))
			for i, v := range bytes {
				is[i] = v
			}
			return is
		},
		"SetResponse": func(args ...gowasm.Value) interface{} {
			if resp == nil {
				return nil
			}
			if len(args) < 1 {
				return gowasm.NewJSError(fmt.Errorf("SetResponse: invalid num of args"))
			}
			if *resp == nil {
				*resp = &plugin.Response{}
			}
			m := args[0].Bytes()
			*err = proto.Unmarshal(m, *resp)
			return nil
		},
		"SetError": func(args ...gowasm.Value) interface{} {
			if len(args) < 1 {
				return gowasm.NewJSError(fmt.Errorf("SetError: invalid num of args"))
			}

			*err = errors.New(args[0].String())
			return nil
		},
		"APIRequest": c.apiRequestFn(sctx, ctx, err),
	}
}

func (c *WASMContractClient) Meta() (meta plugin.Meta, err error) {
	r := gowasm.NewResolver()
	r.StandAlone = false
	handler := c.newContractCallHandler("Meta", nil, nil, nil, nil, &err)
	handler["SetResponse"] = func(args ...gowasm.Value) interface{} {
		if len(args) < 1 {
			return gowasm.NewJSError(fmt.Errorf("SetResponse: invalid num of args"))
		}
		m := args[0].Bytes()
		err = proto.Unmarshal(m, &meta)
		return nil
	}
	r.SetGlobalValue("ContractCallHandler", handler)
	_, err2 := gowasm.RunWASMFileWithResolver(r, c.path, "")
	if err == nil {
		err = err2
	}
	return meta, err
}

func (c *WASMContractClient) Init(ctx plugin.Context, req *plugin.Request) (err error) {
	r := gowasm.NewResolver()
	r.StandAlone = false
	sctx := ctx.(plugin.StaticContext)
	r.SetGlobalValue("ContractCallHandler", c.newContractCallHandler("Init", sctx, ctx, req, nil, &err))
	_, err2 := gowasm.RunWASMFileWithResolver(r, c.path, "")
	if err == nil {
		err = err2
	}
	return err
}

func (c *WASMContractClient) Call(ctx plugin.Context, req *plugin.Request) (resp *plugin.Response, err error) {
	r := gowasm.NewResolver()
	r.StandAlone = false
	sctx := ctx.(plugin.StaticContext)
	r.SetGlobalValue("ContractCallHandler", c.newContractCallHandler("Call", sctx, ctx, req, &resp, &err))
	_, err2 := gowasm.RunWASMFileWithResolver(r, c.path, "")
	if err == nil {
		err = err2
	}
	return
}

func (c *WASMContractClient) StaticCall(ctx plugin.StaticContext, req *plugin.Request) (resp *plugin.Response, err error) {
	r := gowasm.NewResolver()
	r.StandAlone = false
	r.SetGlobalValue("ContractCallHandler", c.newContractCallHandler("StaticCall", ctx, nil, req, &resp, &err))
	_, err2 := gowasm.RunWASMFileWithResolver(r, c.path, "")
	if err == nil {
		err = err2
	}
	return
}

var _ plugin.Contract = &WASMContractClient{}
