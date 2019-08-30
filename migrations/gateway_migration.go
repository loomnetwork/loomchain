// +build gateway

package migrations

import (
	"bytes"

	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
)

func GatewayMigration(ctx *MigrationContext, parameters []byte) error {
	gwMigrationRequest := tgtypes.TransferGatewaySwitchMainnetGatewayRequest{}

	unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(parameters)
	if err := unmarshaler.Unmarshal(buf, &gwMigrationRequest); err != nil {
		return err
	}

	gatewayCtx, err := ctx.ContractContext(gwMigrationRequest.GatewayName)
	if err != nil {
		return err
	}

	if err := gateway.SwitchMainnetGateway(gatewayCtx, &gwMigrationRequest); err != nil {
		return err
	}
	return nil
}
