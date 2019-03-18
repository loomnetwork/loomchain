package chainconfig

import (
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	cctype "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/spf13/cobra"
)

var (
	chainConfigContractName = "chainconfig"
)

func NewChainCfgCommand() *cobra.Command {
	cmd := cli.ContractCallCommand("chainconfig")
	cmd.Use = "chain-cfg"
	cmd.Short = "On-chain configuration CLI"
	cmd.AddCommand(
		EnableFeatureCmd(),
		AddFeatureCmd(),
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
			var resp cctype.EnableFeatureResponse
			err := cli.CallContract(chainConfigContractName, "EnableFeature", &cctype.EnableFeatureRequest{Name: args[0]}, &resp)
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
loom chainconfig add-feature hardfork
`

func AddFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "add-feature <feature name>",
		Short:   "Add feature by feature name",
		Example: enableFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.AddFeatureResponse
			err := cli.CallContract(chainConfigContractName, "AddFeature", &cctype.AddFeatureRequest{Name: args[0]}, &resp)
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
			var resp cctype.GetFeatureResponse
			err := cli.StaticCallContract(chainConfigContractName, "GetFeature", &cctype.GetFeatureRequest{Name: args[0]}, &resp)
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
			var resp cctype.ListFeaturesResponse
			err := cli.StaticCallContract(chainConfigContractName, "ListFeatures", &cctype.ListFeaturesRequest{}, &resp)
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
