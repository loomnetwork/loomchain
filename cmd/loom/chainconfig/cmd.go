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
	cmd := &cobra.Command{
		Use:   "chain-cfg <command>",
		Short: "On-chain configuration CLI",
	}
	cmd.AddCommand(
		EnableFeatureCmd(),
		AddFeatureCmd(),
		GetFeatureCmd(),
		SetParamsCmd(),
		GetParamsCmd(),
		ListFeaturesCmd(),
		FeatureEnabledCmd(),
		RemoveFeatureCmd(),
	)
	return cmd
}

const enableFeatureCmdExample = `
loom chain-cfg enable-feature hardfork multichain
`

func EnableFeatureCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
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
			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "EnableFeature", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const addFeatureCmdExample = `
loom chain-cfg add-feature hardfork multichain --build 866 --no-auto-enable
`

func AddFeatureCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var buildNumber uint64
	var noAutoEnable bool
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
			req := &cctype.AddFeatureRequest{
				Names:       args,
				BuildNumber: buildNumber,
				AutoEnable:  !noAutoEnable,
			}
			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "AddFeature", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	cmdFlags := cmd.Flags()
	cmdFlags.Uint64Var(&buildNumber, "build", 0, "Minimum build number that supports this feature")
	cmdFlags.BoolVar(
		&noAutoEnable,
		"no-auto-enable",
		false,
		"Don't allow validator nodes to auto-enable this feature (operator will have to do so manually)",
	)
	cmd.MarkFlagRequired("build")
	return cmd
}

const setParamsCmdExample = `
loom chain-cfg set-params --vote-threshold 60
loom chain-cfg set-params --block-confirmations 1000
`

func SetParamsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
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
			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "SetParams", request, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	cmdFlags := cmd.Flags()
	cmdFlags.Uint64Var(&voteThreshold, "vote-threshold", 0, "Set vote threshold")
	cmdFlags.Uint64Var(&numBlockConfirmations, "block-confirmations", 0, "Set N block confirmations")
	return cmd
}

const getParamsCmdExample = `
loom chain-cfg get-params
`

func GetParamsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-params",
		Short:   "Get vote-threshold and num-block-confirmation parameters from chainconfig",
		Example: getParamsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.GetParamsResponse
			err := cli.StaticCallContractWithFlags(&flags, chainConfigContractName, "GetParams",
				&cctype.GetParamsRequest{}, &resp)
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getFeatureCmdExample = `
loom chain-cfg get-feature hardfork
`

func GetFeatureCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-feature <feature name>",
		Short:   "Get feature by feature name",
		Example: getFeatureCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.GetFeatureResponse
			err := cli.StaticCallContractWithFlags(&flags, chainConfigContractName, "GetFeature",
				&cctype.GetFeatureRequest{Name: args[0]}, &resp)
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const listFeaturesCmdExample = `
loom chainconfig list-features
`

func ListFeaturesCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-features",
		Short:   "Display all features",
		Example: listFeaturesCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.ListFeaturesResponse
			err := cli.StaticCallContractWithFlags(&flags, chainConfigContractName, "ListFeatures",
				&cctype.ListFeaturesRequest{}, &resp)
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const featureEnabledCmdExample = `
loom chain-cfg feature-enabled hardfork false
`

func FeatureEnabledCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "feature-enabled <feature name> <default value>",
		Short:   "Check if feature is enabled on chain",
		Example: featureEnabledCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp plugintypes.FeatureEnabledResponse
			req := &plugintypes.FeatureEnabledRequest{
				Name:       args[0],
				DefaultVal: false,
			}
			if err := cli.StaticCallContractWithFlags(&flags,
				chainConfigContractName,
				"FeatureEnabled",
				req,
				&resp,
			); err != nil {
				return err
			}
			fmt.Println(resp.Value)
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const removeFeatureCmdExample = `
loom chain-cfg remove-feature tx:migration migration:1
`

func RemoveFeatureCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "remove-feature <feature name 1> ... <feature name N>",
		Short:   "Remove feature by feature name",
		Example: removeFeatureCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range args {
				if name == "" {
					return fmt.Errorf("Invalid feature name")
				}
			}
			var resp cctype.RemoveFeatureRequest
			if err := cli.CallContractWithFlags(&flags,
				chainConfigContractName,
				"RemoveFeature",
				&cctype.RemoveFeatureRequest{
					Names: args,
				},
				&resp,
			); err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

// Utils

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
