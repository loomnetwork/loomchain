package chainconfig

import (
	"fmt"

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
loom chain-cfg enable-feature hardfork multichain
`

func EnableFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "enable-feature <feature name 1> ... <feature name N>",
		Short:   "Enable features by feature names",
		Example: enableFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range args {
				if name == "" {
					return fmt.Errorf("Invalid feature name")
				}
			}
			req := &cctype.EnableFeatureRequest{Names: args}
			err := cli.CallContract(chainConfigContractName, "EnableFeature", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

const addFeatureCmdExample = `
loom chain-cfg add-feature hardfork multichain --build 866
`

func AddFeatureCmd() *cobra.Command {
	var buildNumber uint64
	cmd := &cobra.Command{
		Use:     "add-feature <feature name 1> ... <feature name N>",
		Short:   "Add new feature",
		Example: addFeatureCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range args {
				if name == "" {
					return fmt.Errorf("Invalid feature name")
				}
			}
			req := &cctype.AddFeatureRequest{Names: args, BuildNumber: buildNumber}
			err := cli.CallContract(chainConfigContractName, "AddFeature", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.Uint64Var(&buildNumber, "build", 0, "Minimum build number that supports this feature")
	return cmd
}

const setParamsCmdExample = `
loom chain-cfg set-params --vote-threshold 60
loom chain-cfg set-params --block-confirmations 1000
`

func SetParamsCmd() *cobra.Command {
	voteThreshold := uint64(0)
	numBlockConfirmations := uint64(0)
	cmd := &cobra.Command{
		Use:     "set-params",
		Short:   "Set vote-threshold and num-block-confirmation parameters for chainconfig",
		Example: setParamsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			request := &cctype.SetParamsRequest{
				Params: &cctype.Params{
					VoteThreshold:         voteThreshold,
					NumBlockConfirmations: numBlockConfirmations,
				},
			}
			err := cli.CallContract(chainConfigContractName, "SetParams", request, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.Uint64Var(&voteThreshold, "vote-threshold", 0, "Set vote threshold")
	cmdFlags.Uint64Var(&numBlockConfirmations, "block-confirmations", 0, "Set N block confirmations")
	return cmd
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
		Example: featureEnabledCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp plugintypes.FeatureEnabledResponse
			err := cli.StaticCallContract(chainConfigContractName, "FeatureEnabled", &plugintypes.FeatureEnabledRequest{Name: args[0], DefaultVal: false}, &resp)
			if err != nil {
				return err
			}
			fmt.Println(resp.Value)
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
