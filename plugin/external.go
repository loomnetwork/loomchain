package plugin

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sync"

	extplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom-plugin/plugin"
	"github.com/loomnetwork/loom-plugin/types"
)

type FileNameInfo struct {
	Base    string
	Ext     string
	Version string
}

func (f *FileNameInfo) String() string {
	return f.Base + f.Ext + f.Version
}

var fileInfoRE = regexp.MustCompile("(.+?)(\\.[a-zA-Z]+?)?\\.([0-9\\.]+)")

func parseFileName(name string) (*FileNameInfo, error) {
	groups := fileInfoRE.FindSubmatch([]byte(name))
	if len(groups) < 4 {
		return nil, errors.New("invalid filename format")
	}

	return &FileNameInfo{
		Base:    string(groups[1]),
		Ext:     string(groups[2]),
		Version: string(groups[3]),
	}, nil
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]extplugin.Plugin{
	"contract": &ExternalPlugin{},
}

func isExec(f os.FileInfo) bool {
	return !f.IsDir() && f.Size() > 0 && f.Mode()&0111 > 0
}

func discoverExec(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var execs []string
	for _, file := range files {
		if isExec(file) {
			execs = append(execs, file.Name())
		}
	}

	return execs, nil
}

func clientConfig() *extplugin.ClientConfig {
	return &extplugin.ClientConfig{
		HandshakeConfig: plugin.Handshake,
		Plugins:         PluginMap,
		AllowedProtocols: []extplugin.Protocol{
			extplugin.ProtocolGRPC,
		},
	}
}

func fetchContract(rpcClient extplugin.ClientProtocol) (plugin.Contract, error) {
	raw, err := rpcClient.Dispense("contract")
	if err != nil {
		return nil, err
	}

	return raw.(plugin.Contract), nil
}

func loadExternal(path string) *extplugin.Client {
	cfg := clientConfig()
	cfg.Cmd = exec.Command("sh", "-c", path)
	return extplugin.NewClient(cfg)
}

type ExternalLoader struct {
	Dir     string
	clients map[string]*extplugin.Client
	mu      sync.Mutex
}

var _ Loader = &ExternalLoader{}

func NewExternalLoader(dir string) *ExternalLoader {
	return &ExternalLoader{
		Dir:     dir,
		clients: make(map[string]*extplugin.Client),
	}
}

func (l *ExternalLoader) Kill() {
	var wg sync.WaitGroup
	l.mu.Lock()
	for _, client := range l.clients {
		wg.Add(1)

		go func(client *extplugin.Client) {
			client.Kill()
			wg.Done()
		}(client)
	}
	l.mu.Unlock()
	wg.Wait()
}

func (l *ExternalLoader) LoadContract(name string) (plugin.Contract, error) {
	client, err := l.loadClient(name)
	if err != nil {
		return nil, err
	}

	rpcClient, err := client.Client()
	if err != nil {
		return nil, err
	}

	return fetchContract(rpcClient)
}

func (l *ExternalLoader) loadClient(name string) (*extplugin.Client, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var err error
	client := l.clients[name]
	if client == nil {
		client, err = l.loadClientFull(name)
		if err != nil {
			return nil, err
		}
	}

	l.clients[name] = client
	return client, nil
}

func (l *ExternalLoader) loadClientFull(name string) (*extplugin.Client, error) {
	files, err := discoverExec(l.Dir)
	if err != nil {
		return nil, err
	}

	meta, err := ParseMeta(name)
	if err != nil {
		return nil, err
	}

	var found string
	for _, file := range files {
		info, err := parseFileName(file)
		if err != nil {
			continue
		}

		if info.Base == meta.Name && info.Version == meta.Version {
			found = file
			break
		}
	}

	if found == "" {
		return nil, errors.New("contract not found")
	}

	return loadExternal(path.Join(l.Dir, found)), nil
}

type GRPCAPIServer struct {
	sctx plugin.StaticContext
	ctx  plugin.Context
}

var (
	errVolatileCall = errors.New("calling volatile method from static context")
)

func (s *GRPCAPIServer) Get(ctx context.Context, req *types.GetRequest) (*types.GetResponse, error) {
	return &types.GetResponse{
		Value: s.sctx.Get(req.Key),
	}, nil
}

func (s *GRPCAPIServer) Has(ctx context.Context, req *types.HasRequest) (*types.HasResponse, error) {
	return &types.HasResponse{
		Value: s.sctx.Has(req.Key),
	}, nil
}

func (s *GRPCAPIServer) StaticCall(ctx context.Context, req *types.CallRequest) (*types.CallResponse, error) {
	ret, err := s.sctx.StaticCall(lp.UnmarshalAddressPB(req.Address), req.Input)
	if err != nil {
		return nil, err
	}
	return &types.CallResponse{Output: ret}, nil
}

func (s *GRPCAPIServer) Emit(ctx context.Context, req *types.EmitRequest) (*types.EmitResponse, error) {
	return &types.EmitResponse{}, nil
}

func (s *GRPCAPIServer) Set(ctx context.Context, req *types.SetRequest) (*types.SetResponse, error) {
	if s.ctx == nil {
		return nil, errVolatileCall
	}
	s.ctx.Set(req.Key, req.Value)
	return &types.SetResponse{}, nil
}

func (s *GRPCAPIServer) Delete(ctx context.Context, req *types.DeleteRequest) (*types.DeleteResponse, error) {
	if s.ctx == nil {
		return nil, errVolatileCall
	}
	s.ctx.Delete(req.Key)
	return &types.DeleteResponse{}, nil
}

func (s *GRPCAPIServer) Call(ctx context.Context, req *types.CallRequest) (*types.CallResponse, error) {
	if s.ctx == nil {
		return nil, errVolatileCall
	}
	ret, err := s.ctx.Call(lp.UnmarshalAddressPB(req.Address), req.Input)
	if err != nil {
		return nil, err
	}
	return &types.CallResponse{Output: ret}, nil
}

func bootApiServer(broker *extplugin.GRPCBroker, apiServer *GRPCAPIServer) (*grpc.Server, uint32) {
	var s *grpc.Server

	var wg sync.WaitGroup
	wg.Add(1)
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		defer wg.Done()
		s = grpc.NewServer(opts...)
		types.RegisterAPIServer(s, apiServer)
		return s
	}

	brokerID := broker.NextId()
	go broker.AcceptAndServe(brokerID, serverFunc)
	// TODO: partly copied from example, but does not seem robust as s is set in
	// another goroutine. Does not seem secure either as api server ID can
	// be ignored in plugin.
	wg.Wait()
	return s, brokerID
}

func makeContext(ctx plugin.StaticContext, req *types.Request, apiServer uint32) *types.ContractCallRequest {
	block := ctx.Block()
	msg := ctx.Message()
	return &types.ContractCallRequest{
		Block: &block,
		Message: &types.Message{
			Sender: msg.Sender.MarshalPB(),
		},
		ContractAddress: ctx.ContractAddress().MarshalPB(),
		Request:         req,
		ApiServer:       apiServer,
	}
}

type GRPCContractClient struct {
	broker *extplugin.GRPCBroker
	client types.ContractClient
}

var _ plugin.Contract = &GRPCContractClient{}

func (c *GRPCContractClient) Meta() (types.ContractMeta, error) {
	return types.ContractMeta{}, nil
}

func (c *GRPCContractClient) Init(ctx plugin.Context, req *types.Request) error {
	apiServer := &GRPCAPIServer{
		sctx: ctx,
		ctx:  ctx,
	}
	s, brokerID := bootApiServer(c.broker, apiServer)
	defer s.Stop()

	_, err := c.client.Init(context.TODO(), makeContext(ctx, req, brokerID))
	return err
}

func (c *GRPCContractClient) Call(ctx plugin.Context, req *types.Request) (*types.Response, error) {
	apiServer := &GRPCAPIServer{
		sctx: ctx,
		ctx:  ctx,
	}
	s, brokerID := bootApiServer(c.broker, apiServer)
	defer s.Stop()

	return c.client.Call(context.TODO(), makeContext(ctx, req, brokerID))
}

func (c *GRPCContractClient) StaticCall(ctx plugin.StaticContext, req *types.Request) (*types.Response, error) {
	apiServer := &GRPCAPIServer{
		sctx: ctx,
	}
	s, brokerID := bootApiServer(c.broker, apiServer)
	defer s.Stop()

	return c.client.StaticCall(context.TODO(), makeContext(ctx, req, brokerID))
}

type ExternalPlugin struct {
	extplugin.NetRPCUnsupportedPlugin
}

func (p *ExternalPlugin) GRPCServer(broker *extplugin.GRPCBroker, s *grpc.Server) error {
	return errors.New("not implemented on chain side")
}

func (p *ExternalPlugin) GRPCClient(ctx context.Context, broker *extplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCContractClient{
		broker: broker,
		client: types.NewContractClient(c),
	}, nil
}

var _ extplugin.GRPCPlugin = &ExternalPlugin{}
