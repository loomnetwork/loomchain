// +build gateway

package migrations

import (
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
)

func GatewayMigration(ctx *MigrationContext) error {
	gatewayCtx, err := ctx.ContractContext("gateway")
	if err != nil {
		return err
	}
	gateway.SetSignedReceiptsToUnsigned(gatewayCtx)
	return nil
}
