package chainconfig

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	cctype "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/config"
	plugintypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/ed25519"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
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
		SetSettingCmd(),
		ListPendingActionsCmd(),
		ChainConfigCmd(),
		SetValidatorInfoCmd(),
		GetValidatorInfoCmd(),
		ListValidatorsInfoCmd(),
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

const setValidatorInfoCmdExample = `
loom chain-cfg set-validator-info --build 1000 --key path/to/private_key
`

func SetValidatorInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	buildNumber := uint64(0)
	cmd := &cobra.Command{
		Use:     "set-validator-info",
		Short:   "Set validator information",
		Example: setValidatorInfoCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			request := &cctype.SetValidatorInfoRequest{
				BuildNumber: buildNumber,
			}
			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "SetValidatorInfo", request, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	cmdFlags := cmd.Flags()
	cmdFlags.Uint64Var(&buildNumber, "build", 0, "Specifies the build number the validator is running")
	return cmd
}

const getValidatorInfoCmdExample = `
loom chain-cfg get-validator-info 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func GetValidatorInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-validator-info",
		Short:   "Get validator information",
		Example: getValidatorInfoCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.GetValidatorInfoResponse
			addr, _ := cli.ParseAddress(args[0], flags.ChainID)

			if err := cli.StaticCallContractWithFlags(&flags,
				chainConfigContractName,
				"GetValidatorInfo",
				&cctype.GetValidatorInfoRequest{Address: addr.MarshalPB()},
				&resp,
			); err != nil {
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

const listPendingActionsCmdExample = `
loom chain-cfg list-pending-actions
`

func ListPendingActionsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-pending-actions",
		Short:   "Show all pending actions to change settings",
		Example: listPendingActionsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.ListPendingActionsResponse
			err := cli.StaticCallContractWithFlags(
				&flags, chainConfigContractName,
				"ListPendingActions", &cctype.ListPendingActionsRequest{}, &resp,
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

const chainConfigCmdExample = `
loom chain-cfg config
`

func ChainConfigCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Get on-chain config",
		Example: chainConfigCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.ChainConfigResponse
			err := cli.StaticCallContractWithFlags(
				&flags, chainConfigContractName,
				"ChainConfig", &cctype.ChainConfigRequest{}, &resp,
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

const setSettingCmdExample = `
loom chain-cfg set-setting AppStore.NumEvmKeysToPrune --value 100 --build 1200 -k private_key
`

func SetSettingCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var value string
	var buildNumber uint64
	cmd := &cobra.Command{
		Use:     "set-setting <config name>",
		Short:   "Set setting",
		Example: setSettingCmdExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" || value == "" {
				return fmt.Errorf("invalid config key")
			}

			// validate config setting
			defaultConfig := config.DefaultConfig()
			if err := config.SetConfigSetting(defaultConfig, args[0], value); err != nil {
				return err
			}

			req := &cctype.SetSettingRequest{
				Name:        args[0],
				Value:       value,
				BuildNumber: buildNumber,
			}

			err := cli.CallContractWithFlags(&flags, chainConfigContractName, "SetSetting", req, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&value, "value", "", "Value of config setting")
	cmdFlags.Uint64Var(&buildNumber, "build", 0, "Minimum build number required for this change to apply")
	cmd.MarkFlagRequired("value")
	cmd.MarkFlagRequired("build")
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const listValidatorsInfoCmdExample = `
loom chain-cfg list-validators
`

func ListValidatorsInfoCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	var showAll bool
	cmd := &cobra.Command{
		Use:     "list-validators",
		Short:   "Show info stored for each validator",
		Example: listValidatorsInfoCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp cctype.ListValidatorsInfoResponse
			err := cli.StaticCallContractWithFlags(
				&flags, chainConfigContractName, "ListValidatorsInfo", &cctype.ListValidatorsInfoRequest{}, &resp,
			)
			if err != nil {
				return err
			}
			var respDPOS dposv3.ListCandidatesResponse
			err = cli.StaticCallContractWithFlags(
				&flags, "dposV3", "ListCandidates", &dposv3.ListCandidatesRequest{}, &respDPOS,
			)
			if err != nil {
				return err
			}

			var rawJSON json.RawMessage
			rpcclient := client.NewJSONRPCClient(flags.URI + "/rpc")
			err = rpcclient.Call("validators", map[string]interface{}{}, "11", &rawJSON)
			if err != nil {
				return err
			}
			cdc := amino.NewCodec()
			coretypes.RegisterAmino(cdc)
			var rpcResult coretypes.ResultValidators
			if err := cdc.UnmarshalJSON(rawJSON, &rpcResult); err != nil {
				return err
			}

			powerSum := int64(0)
			activeValidatorList := make(map[string]int64, len(rpcResult.Validators))
			for _, v := range rpcResult.Validators {
				pubKey := [ed25519.PubKeyEd25519Size]byte(v.PubKey.(ed25519.PubKeyEd25519))
				activeValidatorList[loom.LocalAddressFromPublicKey(pubKey[:]).String()] = v.VotingPower
				powerSum += v.VotingPower
			}

			type maxLength struct {
				Name        int
				Validator   int
				BuildNumber int
				UpdateAt    int
				Status      int
				Power       int
			}

			ml := maxLength{Name: 20, Validator: 42, BuildNumber: 5, UpdateAt: 29, Status: 6, Power: 5}

			sort.Slice(resp.Validators[:], func(i, j int) bool {
				return resp.Validators[i].BuildNumber < resp.Validators[j].BuildNumber
			})

			nameList := make(map[string]string)
			for _, c := range respDPOS.Candidates {
				n := c.Candidate.GetName()
				nameList[loom.UnmarshalAddressPB(c.Candidate.Address).Local.String()] = n
				if ml.Name < len(n) {
					ml.Name = len(n)
				}
			}

			fmt.Printf(
				"%-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n", ml.Name, "name", ml.Validator, "validator",
				ml.BuildNumber, "build", ml.Status, "active", ml.Power, "power", ml.UpdateAt, "Last Update")
			fmt.Printf(
				strings.Repeat("-", ml.Name+ml.Validator+ml.BuildNumber+ml.Status+ml.Power+ml.UpdateAt+16) + "\n")
			for _, v := range resp.Validators {
				validatorAddr := v.Address.Local.String()
				if !showAll && activeValidatorList[v.Address.Local.String()] == 0 {
					continue
				}
				fmt.Printf(
					"%-*s | %-*s | %-*d | %-*v | %*.2f | %-*s |\n",
					ml.Name, nameList[validatorAddr],
					ml.Validator, validatorAddr,
					ml.BuildNumber, v.BuildNumber,
					ml.Status, activeValidatorList[validatorAddr] > 0,
					ml.Power, float64(activeValidatorList[validatorAddr])/float64(powerSum)*float64(100),
					ml.UpdateAt, time.Unix(int64(v.UpdatedAt), 0).UTC(),
				)
			}

			counters := make(map[uint64]int)
			for _, validator := range resp.Validators {
				if !showAll && activeValidatorList[validator.Address.Local.String()] == 0 {
					continue
				}
				counters[validator.BuildNumber]++
			}
			fmt.Printf(
				"\n%-*s| %-*s | \n", 11, "BuildNumber", 10, "Percentage")
			fmt.Printf(
				strings.Repeat("-", 25) + "\n")

			for k, v := range counters {
				if showAll {
					fmt.Printf("%-*d | %-*d  | \n", 10, k, 9, v*100/len(resp.Validators))
				} else {
					fmt.Printf("%-*d | %-*d  | \n", 10, k, 9, v*100/len(activeValidatorList))
				}
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.BoolVar(&showAll, "all", false, "Show both active and inactive validators")
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
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

type ByBuild []cctype.ValidatorInfo

func (a ByBuild) Len() int           { return len(a) }
func (a ByBuild) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByBuild) Less(i, j int) bool { return a[i].BuildNumber < a[j].BuildNumber }
