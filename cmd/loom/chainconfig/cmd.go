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
		SetFeatureCmd(),
		GetFeatureCmd(),
		ListFeaturesCmd(),
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
			err := cli.CallContract(ChainConfigContractName, "EnableFeature", &chainconfig.EnableFeatureRequest{Name: args[0]}, &resp)
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

const setFeatureCmdExample = `
loom chainconfig set-feature hardfork
`

func SetFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "set-feature <feature name>",
		Short:   "Set feature by feature name",
		Example: enableFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp chainconfig.SetFeatureResponse
			err := cli.CallContract(ChainConfigContractName, "SetFeature", &chainconfig.SetFeatureRequest{Name: args[0]}, &resp)
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

const getFeatureCmdExample = `
loom chainconfig get-feature hardfork
`

func GetFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get-feature <feature name>",
		Short:   "Get feature by feature name",
		Example: getFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp chainconfig.GetFeatureResponse
			err := cli.StaticCallContract(ChainConfigContractName, "GetFeature", &chainconfig.GetFeatureRequest{Name: args[0]}, &resp)
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

const listFeaturesCmdExample = `
loom chainconfig list-features
`

func ListFeaturesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list-features",
		Short:   "Display all features",
		Example: listFeaturesCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp chainconfig.ListFeaturesResponse
			err := cli.StaticCallContract(ChainConfigContractName, "ListFeatures", &chainconfig.ListFeaturesRequest{}, &resp)
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
