package chainconfig

import (
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/spf13/cobra"
)

var (
	ChainConfigContractName = "chainconfig"
)

func NewChainconfigCommand() *cobra.Command {
	cmd := cli.ContractCallCommand("chainconfig")
	cmd.Use = "chainconfig"
	cmd.Short = "Run chainconfig commands"
	cmd.AddCommand(
		EnableFeatureCmd(),
	)
	return cmd
}

const enableFeatureCmdExample = `
loom chainconfig enable-feature hardfork
`

func EnableFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "enable-feature <feature name>",
		Short:   "Enable feature by feature name",
		Example: enableFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp chainconfig.EnableFeatureResponse
			err := cli.CallContract(ChainConfigContractName, "EnableFeature", &chainconfig.EnableFeatureRequest{Name: "Hello"}, &resp)
			if err != nil {
				return err
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}

// Utils

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
