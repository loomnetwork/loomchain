// +build !evm

package gateway

import (
	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	cmd := newRootCommand()
	return cmd
}
