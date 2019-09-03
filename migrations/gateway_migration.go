// +build gateway

package migrations

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
)

func GatewayMigration(ctx *MigrationContext, parameters []byte) error {
	gwMigrationRequest := tgtypes.TransferGatewaySwitchMainnetGatewayRequest{}

	buf := bytes.NewBuffer(parameters)
	err := jsonpb.Unmarshal(buf, &gwMigrationRequest)
	if err != nil {
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
