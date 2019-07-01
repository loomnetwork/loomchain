package chainconfig

import (
	"fmt"
	"strconv"
	"strings"

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
		SetConfigCmd(),
		AddConfigCmd(),
		ListConfigsCmd(),
		GetConfigCmd(),
		ChainConfigCmd(),
		RemoveConfigCmd(),
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

			type maxLength struct {
				Name        int
				Status      int
				Validators  int
				Height      int
				Percentage  int
				BuildNumber int
			}

			ml := maxLength{Name: 4, Status: 7, Validators: 10, Height: 6, Percentage: 6, BuildNumber: 5}
			for _, value := range resp.Features {
				if len(value.Name) > ml.Name {
					ml.Name = len(value.Name)
				}
				if uintLength(value.BlockHeight) > ml.Height {
					ml.Height = uintLength(value.BlockHeight)
				}
			}
			fmt.Printf(
				"%-*s | %-*s | %-*s | %-*s | %-*s | %-*s\n", ml.Name,
				"name", ml.Status, "status", ml.Validators, "validators",
				ml.Height, "height", ml.Percentage, "vote %", ml.BuildNumber, "build")
			fmt.Printf(
				strings.Repeat("-", ml.Name+ml.Status+ml.Validators+
					ml.Height+ml.Percentage+ml.BuildNumber+15) + "\n")
			for _, value := range resp.Features {
				fmt.Printf("%-*s | %-*s | %-*d | %-*d | %-*d | %-*d\n",
					ml.Name, value.Name, ml.Status, value.Status,
					ml.Validators, len(value.Validators), ml.Height,
					value.BlockHeight, ml.Percentage, value.Percentage,
					ml.BuildNumber, value.BuildNumber)
			}
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

const addConfigCmdExample = `
loom chain-cfg add-config dpos.feeFloor --build 866 --vote-threshold 70
`

func AddConfigCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var buildNumber uint64
	var voteThreshold uint64
	var value string
	cmd := &cobra.Command{
		Use:     "add-config <config name 1> ... <config name N>",
		Short:   "Add new config",
		Example: addConfigCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range args {
				if name == "" {
					return fmt.Errorf("Invalid config name")
				}
			}
			req := &cctype.AddConfigRequest{
				Names:         args,
				BuildNumber:   buildNumber,
				VoteThreshold: voteThreshold,
				Value:         value,
			}
			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "AddConfig", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.Uint64Var(&buildNumber, "build", 0, "Minimum build number that supports this config")
	cmdFlags.Uint64Var(&voteThreshold, "vote-threshold", 0, "Set vote threshold")
	cmdFlags.StringVar(&value, "value", "", "Set value of the config")
	cmd.MarkFlagRequired("build")
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const listConfigsCmdExample = `
loom chainconfig list-configs
`

func ListConfigsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-configs",
		Short:   "Display all configs",
		Example: listConfigsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.ListConfigsResponse
			err := cli.StaticCallContractWithFlags(&flags, chainConfigContractName,
				"ListConfigs", &cctype.ListConfigsRequest{}, &resp)
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
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getConfigCmdExample = `
loom chain-cfg get-config dpos.feeFloor
`

func GetConfigCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-config <config name>",
		Short:   "Get config by config name",
		Example: getConfigCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.GetConfigResponse
			err := cli.StaticCallContractWithFlags(
				&flags, chainConfigContractName,
				"GetConfig", &cctype.GetConfigRequest{Name: args[0]}, &resp,
			)
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
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const setConfigCmdExample = `
loom chain-cfg set-config dpos.feeFloor --value 15
`

func SetConfigCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var value string
	cmd := &cobra.Command{
		Use:     "set-config <config name>",
		Short:   "Set config setting",
		Example: setConfigCmdExample,
		Args:    cobra.RangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" || value == "" {
				return fmt.Errorf("Invalid config name")
			}
			req := &cctype.SetConfigRequest{
				Name:  args[0],
				Value: value,
			}
			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "SetConfig", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&value, "value", "", "Set value of config")
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const chainconfigCmdExample = `
loom chain-cfg config-value dpos.feeFloor
`

func ChainConfigCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "config-value <config name>",
		Short:   "Get the activated value on chain of a config",
		Example: chainconfigCmdExample,
		Args:    cobra.RangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("Invalid config name")
			}
			var resp cctype.ConfigValueResponse
			req := &cctype.ConfigValueResponse{
				Name: args[0],
			}
			err := cli.StaticCallContractWithFlags(&flags, chainConfigContractName, "ConfigValue", req, &resp)
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
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const removeConfigCmdExample = `
loom chain-cfg remove-config dpos.feeFloor dpos.lockTime
`

func RemoveConfigCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "remove-config <config name 1> ... <config name N>",
		Short:   "Remove config by config name",
		Example: removeConfigCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range args {
				if name == "" {
					return fmt.Errorf("Invalid feature name")
				}
			}
			if err := cli.CallContractWithFlags(
				&flags,
				chainConfigContractName,
				"RemoveConfig",
				&cctype.RemoveConfigRequest{
					Names: args,
				},
				nil,
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

func uintLength(n uint64) int {
	str := strconv.FormatUint(n, 10)
	return len(str)
}
