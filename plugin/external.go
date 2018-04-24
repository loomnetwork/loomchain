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

func loadExternal(path string) *extplugin.Client {
	return extplugin.NewClient(&extplugin.ClientConfig{
		HandshakeConfig: plugin.Handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command("sh", "-c", path),
		AllowedProtocols: []extplugin.Protocol{
			extplugin.ProtocolGRPC,
		},
	})
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

	raw, err := rpcClient.Dispense("contract")
	if err != nil {
		return nil, err
	}

	return raw.(plugin.Contract), nil
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
	return &types.CallResponse{}, nil
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
	return &types.CallResponse{}, nil
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

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		types.RegisterAPIServer(s, apiServer)
		return s
	}

	brokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(brokerID, serverFunc)
	// TODO: copied from example, but does not seem robust as s is set in
	// another goroutine. Does not seem secure either as api server ID can
	// be ignored in plugin.
	defer s.Stop()

	init := &types.ContractCallRequest{
		Block:     &types.BlockHeader{},
		Message:   &types.Message{},
		Request:   req,
		ApiServer: brokerID,
	}
	_, err := c.client.Init(context.TODO(), init)
	return err
}

func (c *GRPCContractClient) Call(ctx plugin.Context, req *types.Request) (*types.Response, error) {
	return nil, nil
}

func (c *GRPCContractClient) StaticCall(ctx plugin.StaticContext, req *types.Request) (*types.Response, error) {
	return nil, nil
}

type ExternalPlugin struct {
	extplugin.NetRPCUnsupportedPlugin
}

func (p *ExternalPlugin) GRPCServer(broker *extplugin.GRPCBroker, s *grpc.Server) error {
	return errors.New("not implemented")
}

func (p *ExternalPlugin) GRPCClient(ctx context.Context, broker *extplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCContractClient{
		broker: broker,
		client: types.NewContractClient(c),
	}, nil
}

var _ extplugin.GRPCPlugin = &ExternalPlugin{}
