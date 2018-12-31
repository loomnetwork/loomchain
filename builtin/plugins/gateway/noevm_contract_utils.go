// +build !evm

package gateway

import (
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
)

type (
	InitRequest = tgtypes.TransferGatewayInitRequest
)
