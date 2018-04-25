package plugin

import (
	"context"
	"testing"

	extplugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/loomnetwork/loom-plugin/plugin"
)

type EmptyContract struct {
}

var _ plugin.Contract = &EmptyContract{}

func (c *EmptyContract) Meta() (plugin.Meta, error) {
	return plugin.Meta{}, nil
}

func (c *EmptyContract) Init(ctx plugin.Context, req *plugin.Request) error {
	return nil
}

func (c *EmptyContract) Call(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return nil, nil
}

func (c *EmptyContract) StaticCall(ctx plugin.StaticContext, req *plugin.Request) (*plugin.Response, error) {
	return nil, nil
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

type PluginTestSuite struct {
	suite.Suite
}

func (s *PluginTestSuite) TestMeta() {
	impl := &EmptyContract{}
	rpcClient, rpcServer := extplugin.TestPluginGRPCConn(s.T(), map[string]extplugin.Plugin{
		"contract": NewCrossExternalPlugin(impl),
	})
	defer rpcClient.Close()
	defer rpcServer.Stop()

	contract, err := fetchContract(rpcClient)
	require.Nil(s.T(), err)

	_, err = contract.Meta()
	assert.Nil(s.T(), err)
}

func TestPluginSuite(t *testing.T) {
	suite.Run(t, new(PluginTestSuite))
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
