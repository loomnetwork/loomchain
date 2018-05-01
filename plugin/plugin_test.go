package plugin

import (
	"context"
	"testing"

	extplugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/loomnetwork/go-loom/plugin"
)

type MockContract struct {
	meta       func() (plugin.Meta, error)
	init       func(ctx plugin.Context, req *plugin.Request) error
	call       func(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error)
	staticCall func(ctx plugin.StaticContext, req *plugin.Request) (*plugin.Response, error)
}

var _ plugin.Contract = &MockContract{}

func (c *MockContract) Meta() (plugin.Meta, error) {
	return c.meta()
}

func (c *MockContract) Init(ctx plugin.Context, req *plugin.Request) error {
	return c.init(ctx, req)
}

func (c *MockContract) Call(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return c.call(ctx, req)
}

func (c *MockContract) StaticCall(ctx plugin.StaticContext, req *plugin.Request) (*plugin.Response, error) {
	return c.staticCall(ctx, req)
}

type CrossExternalPlugin struct {
	extplugin.NetRPCUnsupportedPlugin
	ClientPlugin *ExternalPlugin
	ServerPlugin *plugin.ExternalPlugin
}

func NewCrossExternalPlugin(impl plugin.Contract) *CrossExternalPlugin {
	return &CrossExternalPlugin{
		ClientPlugin: &ExternalPlugin{},
		ServerPlugin: &plugin.ExternalPlugin{
			Impl: impl,
		},
	}
}

func (p *CrossExternalPlugin) GRPCServer(broker *extplugin.GRPCBroker, s *grpc.Server) error {
	return p.ServerPlugin.GRPCServer(broker, s)
}

func (p *CrossExternalPlugin) GRPCClient(ctx context.Context, broker *extplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return p.ClientPlugin.GRPCClient(ctx, broker, c)
}

type closeableContract interface {
	plugin.Contract
	Close() error
}

type contractWrapperFn func(t *testing.T, impl plugin.Contract) closeableContract

type externalContract struct {
	plugin.Contract
	rpcClient *extplugin.GRPCClient
	rpcServer *extplugin.GRPCServer
}

func (c *externalContract) Close() error {
	err := c.rpcClient.Close()
	c.rpcServer.Stop()
	return err
}

func externalWrap(t *testing.T, impl plugin.Contract) closeableContract {
	rpcClient, rpcServer := extplugin.TestPluginGRPCConn(t, map[string]extplugin.Plugin{
		"contract": NewCrossExternalPlugin(impl),
	})

	contract, err := fetchContract(rpcClient)
	ret := &externalContract{
		Contract:  contract,
		rpcClient: rpcClient,
		rpcServer: rpcServer,
	}
	if err != nil {
		ret.Close()
	}
	require.Nil(t, err)
	return ret
}

type PluginTestSuite struct {
	contractWrapperFn
	suite.Suite
}

func (s *PluginTestSuite) TestMeta() {
	impl := &MockContract{
		meta: func() (plugin.Meta, error) {
			return plugin.Meta{
				Name:    "foo",
				Version: "1.0.0",
			}, nil
		},
	}
	contract := s.contractWrapperFn(s.T(), impl)
	defer contract.Close()

	meta, err := contract.Meta()
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "foo", meta.Name)
	assert.Equal(s.T(), "1.0.0", meta.Version)
}

func (s *PluginTestSuite) TestInit() {
	s.T().Skip("skip broken test for now")

	impl := &MockContract{
		init: func(ctx plugin.Context, req *plugin.Request) error {
			ctx.Set([]byte("foo"), []byte("bar"))
			return nil
		},
	}
	contract := s.contractWrapperFn(s.T(), impl)
	defer contract.Close()

	ctx := plugin.CreateFakeContext()
	err := contract.Init(ctx, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "bar", string(ctx.Get([]byte("foo"))))
}

func TestPluginSuite(t *testing.T) {
	suite.Run(t, &PluginTestSuite{
		contractWrapperFn: externalWrap,
	})
}

func TestParseFileName(t *testing.T) {
	info, err := parseFileName("hello.so.1.2.3")
	require.Nil(t, err)

	assert.Equal(t, "hello", info.Base)
	assert.Equal(t, ".so", info.Ext)
	assert.Equal(t, "1.2.3", info.Version)

	info, err = parseFileName("hello.1.2.3")
	require.Nil(t, err)

	assert.Equal(t, "hello", info.Base)
	assert.Equal(t, "", info.Ext)
	assert.Equal(t, "1.2.3", info.Version)

	_, err = parseFileName("hello.py")
	require.NotNil(t, err)
}
