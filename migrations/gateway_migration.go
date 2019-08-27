// +build gateway

package migrations

import (
	"github.com/gogo/protobuf/proto"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
)

func GatewayMigration(ctx *MigrationContext, parameters []byte) error {

	gwMigrationRequest := tgtypes.TransferGatewaySwitchMainnetGatewayRequest{}
	err := proto.Unmarshal(parameters, &gwMigrationRequest)
	if err != nil {
		return err
	}

	gatewayCtx, err := ctx.ContractContext(gwMigrationRequest.GatewayName)
	if err != nil {
		return err
	}

	gateway.SwitchMainnetGateway(gatewayCtx, gwMigrationRequest)
	return nil
}
