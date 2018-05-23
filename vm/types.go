package vm

import (
	lvm "github.com/loomnetwork/go-loom/vm"
)

type VMType = lvm.VMType

const (
	VMType_PLUGIN VMType = lvm.VMType_PLUGIN
	VMType_EVM    VMType = lvm.VMType_EVM
)

var VMType_value = lvm.VMType_value

type MessageTx = lvm.MessageTx
type DeployTx = lvm.DeployTx
type DeployResponse = lvm.DeployResponse
type CallTx = lvm.CallTx
type Event = lvm.Event
type EvmTxReceipt = lvm.EvmTxReceipt
