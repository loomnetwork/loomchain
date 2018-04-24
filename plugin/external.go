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

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	lp "github.com/loomnetwork/loom-plugin/plugin"
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
var PluginMap = map[string]plugin.Plugin{
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

func loadExternal(path string) *plugin.Client {
	return plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: lp.Handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command("sh", "-c", path),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	})
}

type ExternalLoader struct {
	Dir     string
	clients map[string]*plugin.Client
	mu      sync.Mutex
}

var _ Loader = &ExternalLoader{}

func NewExternalLoader(dir string) *ExternalLoader {
	return &ExternalLoader{
		Dir:     dir,
		clients: make(map[string]*plugin.Client),
	}
}

func (l *ExternalLoader) Kill() {
	var wg sync.WaitGroup
	l.mu.Lock()
	for _, client := range l.clients {
		wg.Add(1)

		go func(client *plugin.Client) {
			client.Kill()
			wg.Done()
		}(client)
	}
	l.mu.Unlock()
	wg.Wait()
}

func (l *ExternalLoader) LoadContract(name string) (lp.Contract, error) {
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

	return raw.(lp.Contract), nil
}

func (l *ExternalLoader) loadClient(name string) (*plugin.Client, error) {
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

func (l *ExternalLoader) loadClientFull(name string) (*plugin.Client, error) {
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
	ctx lp.Context
}

func (s *GRPCAPIServer) Get(ctx context.Context, req *types.GetRequest) (*types.GetResponse, error) {
	return &types.GetResponse{
		Value: s.ctx.Get(req.Key),
	}, nil
}

func (s *GRPCAPIServer) Has(ctx context.Context, req *types.HasRequest) (*types.HasResponse, error) {
	return &types.HasResponse{
		Value: s.ctx.Has(req.Key),
	}, nil
}

func (s *GRPCAPIServer) StaticCall(ctx context.Context, req *types.CallRequest) (*types.CallResponse, error) {
	return &types.CallResponse{}, nil
}

func (s *GRPCAPIServer) Emit(ctx context.Context, req *types.EmitRequest) (*types.EmitResponse, error) {
	return &types.EmitResponse{}, nil
}

func (s *GRPCAPIServer) Set(ctx context.Context, req *types.SetRequest) (*types.SetResponse, error) {
	return &types.SetResponse{}, nil
}

func (s *GRPCAPIServer) Delete(ctx context.Context, req *types.DeleteRequest) (*types.DeleteResponse, error) {
	return &types.DeleteResponse{}, nil
}

func (s *GRPCAPIServer) Call(ctx context.Context, req *types.CallRequest) (*types.CallResponse, error) {
	return &types.CallResponse{}, nil
}

type GRPCContractClient struct {
	broker *plugin.GRPCBroker
	client types.ContractClient
}

var _ lp.Contract = &GRPCContractClient{}

func (c *GRPCContractClient) Meta() (types.ContractMeta, error) {
	return types.ContractMeta{}, nil
}

func (c *GRPCContractClient) Init(ctx lp.Context, req *types.Request) error {
	apiServer := &GRPCAPIServer{ctx: ctx}

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		types.RegisterAPIServer(s, apiServer)
		return s
	}

	brokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(brokerID, serverFunc)

	init := &types.ContractCallRequest{
		Block:     &types.BlockHeader{},
		Message:   &types.Message{},
		Request:   req,
		ApiServer: brokerID,
	}
	_, err := c.client.Init(context.TODO(), init)

	// TODO: copied from example, but does not seem robust as s is set in
	// another goroutine. Does not seem secure either as broker ID can
	// be ignored in plugin.
	s.Stop()
	return err
}

func (c *GRPCContractClient) Call(ctx lp.Context, req *types.Request) (*types.Response, error) {
	return nil, nil
}

func (c *GRPCContractClient) StaticCall(ctx lp.StaticContext, req *types.Request) (*types.Response, error) {
	return nil, nil
}

type ExternalPlugin struct {
	lp.ExternalPlugin
}

func (p *ExternalPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCContractClient{
		broker: broker,
		client: types.NewContractClient(c),
	}, nil
}

var _ plugin.GRPCPlugin = &ExternalPlugin{}
