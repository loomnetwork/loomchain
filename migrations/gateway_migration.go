// +build gateway

package migrations

import (
	"bytes"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
	"github.com/pkg/errors"
)

func GatewayMigration(ctx *MigrationContext, parameters []byte) error {
	req := tgtypes.TransferGatewaySwitchMainnetGatewayRequest{}
	if err := jsonpb.Unmarshal(bytes.NewBuffer(parameters), &req); err != nil {
		return errors.Wrap(err, "failed to unmarshal migration parameters")
	}

	if len(strings.TrimSpace(req.GatewayName)) == 0 {
		return errors.New("missing gateway name in migration parameters")
	}

	gatewayCtx, err := ctx.ContractContext(req.GatewayName)
	if err != nil {
		return err
	}

	return gateway.SwitchMainnetGateway(gatewayCtx, &req)
}
