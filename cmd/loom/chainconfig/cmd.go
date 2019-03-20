package chainconfig

import (
	"fmt"
	"strconv"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	cctype "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/cli"
	plugintypes "github.com/loomnetwork/go-loom/plugin/types"
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
		SetParamsCmd(),
		GetParamsCmd(),
		ListFeaturesCmd(),
		FeatureEnabledCmd(),
	)
	return cmd
}

const enableFeatureCmdExample = `
loom chain-cfg enable-feature hardfork
`

func EnableFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "enable-feature <feature name>",
		Short:   "Enable feature by feature name",
		Example: enableFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cli.CallContract(chainConfigContractName, "EnableFeature", &cctype.EnableFeatureRequest{Name: args[0]}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

const addFeatureCmdExample = `
loom chain-cfg add-feature hardfork
`

func AddFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "add-feature <feature name>",
		Short:   "Add feature by feature name",
		Example: addFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cli.CallContract(chainConfigContractName, "AddFeature", &cctype.AddFeatureRequest{Name: args[0]}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

const setParamsCmdExample = `
loom chain-cfg set-params 66 10
`

func SetParamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "set-params <VoteThreshold> <NumBlockConfirmation>",
		Short:   "Set vote-threshold and num-block-confirmation parameters for chainconfig",
		Example: setParamsCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			voteThreshold, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return err
			}
			numBlockConfirmations, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return err
			}
			request := &cctype.SetParamsRequest{
				Params: &cctype.Params{
					VoteThreshold:         voteThreshold,
					NumBlockConfirmations: numBlockConfirmations,
				},
			}
			err = cli.CallContract(chainConfigContractName, "SetParams", request, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

const getParamsCmdExample = `
loom chain-cfg get-params
`

func GetParamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get-params",
		Short:   "Get vote-threshold and num-block-confirmation parameters from chainconfig",
		Example: getParamsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.GetParamsResponse
			err := cli.StaticCallContract(chainConfigContractName, "GetParams", &cctype.GetParamsRequest{}, &resp)
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
loom chain-cfg get-feature hardfork
`

func GetFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get-feature <feature name>",
		Short:   "Get feature by feature name",
		Example: getFeatureCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const featureEnabledCmdExample = `
loom chain-cfg feature-enabled hardfork false
`

func FeatureEnabledCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "feature-enabled <feature name> <default value>",
		Short:   "Check if feature is enabled on chain",
		Example: getFeatureCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp plugintypes.FeatureEnabledResponse
			err := cli.StaticCallContract(chainConfigContractName, "FeatureEnabled", &plugintypes.FeatureEnabledRequest{Name: args[0], DefaultVal: false}, &resp)
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
