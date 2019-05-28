package staking

import (
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/builtin/commands"
	"github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/builtin/types/dposv2"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/spf13/cobra"
)

const (
	dPOSV2ContractName        = "dposV2"
	addressMapperContractName = "addressmapper"
)

func NewStakingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staking <command>",
		Short: "Run staking commands",
	}
	cmd.AddCommand(
		ListAllDelegationsCmd(),
		GetMappingCmd(),
		ListMappingCmd(),
		ListDelegationsCmd(),
		ListValidatorsCmd(),
		TotalDelegationCmd(),
		CheckDelegationsCmd(),
		GetBalanceCmd(),
		WithdrawalReceiptCmd(),
		CheckAllDelegationsCmd(),
		ListCandidatesCmd(),
	)
	return cmd
}

func ListAllDelegationsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-all-delegations",
		Short: "display the all delegations",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListAllDelegationsResponse
			err := cli.StaticCallContractWithFlags(&flags, dPOSV2ContractName, "ListAllDelegations",
				&dposv2.ListAllDelegationsRequest{}, &resp)
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

const listDelegationsCmdExample = `
loom staking list-delegations 0x0ca3d6bf201ce53c7ddc3cb397ae33a68ed4a328
`

func ListDelegationsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-delegations <validator hex address>",
		Short:   "List a validator's delegations and delegation total",
		Example: listDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			var resp dposv2.ListDelegationsResponse
			err = cli.StaticCallContractWithFlags(&flags,
				dPOSV2ContractName, "ListDelegations",
				&dposv2.ListDelegationsRequest{Candidate: addr.MarshalPB()}, &resp,
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

func ListValidatorsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-validators",
		Short: "List the current validators",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListValidatorsResponseV2
			// nolint:lll
			err := cli.StaticCallContractWithFlags(&flags, dPOSV2ContractName, "ListValidators", &dposv2.ListValidatorsRequestV2{},
				&resp)
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

const totalDelegationCmdExample = `
# Get total delegations using a DAppChain address
loom staking total-delegation 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

# Get total delegations using an Ethereum address
loom staking total-delegation eth:0x751481F4db7240f4d5ab5d8c3A5F6F099C824863
`

func TotalDelegationCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "total-delegation <delegator address>",
		Short:   "Display total staking amount that has delegated to all validators",
		Example: totalDelegationCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			var resp dposv2.TotalDelegationResponse
			err = cli.StaticCallContractWithFlags(&flags,
				dPOSV2ContractName, "TotalDelegation",
				&dposv2.TotalDelegationRequest{DelegatorAddress: addr.MarshalPB()}, &resp,
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const checkDelegationsCmdExample = `
loom staking check-delegation 0x0ca3d6bf201ce53c7ddc3cb397ae33a68ed4a328 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863
`

func CheckDelegationsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "check-delegation (validator address) (delegator address)",
		Short:   "Check delegation to a particular validator",
		Example: checkDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.CheckDelegationResponseV2
			validatorAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			delegatorAddress, err := cli.ResolveAccountAddress(args[1], &flags)
			if err != nil {
				return err
			}
			err = cli.StaticCallContractWithFlags(&flags,
				dPOSV2ContractName, "CheckDelegation",
				&dposv2.CheckDelegationRequestV2{
					ValidatorAddress: validatorAddress.MarshalPB(),
					DelegatorAddress: delegatorAddress.MarshalPB(),
				}, &resp,
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getMappingCmdExample = `
# Get mapping address using a DAppChain address
loom staking get-mapping 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

# Get mapping using an Ethereum address
loom staking get-mapping eth:0x0BE2BC95ea604a5ac4ECcE0F8570fe58bC9C320A
`

func GetMappingCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-mapping",
		Short:   "Get mapping address",
		Example: getMappingCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.AddressMapperGetMappingResponse
			from, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			err = cli.StaticCallContractWithFlags(&flags,
				addressMapperContractName, "GetMapping",
				&address_mapper.AddressMapperGetMappingRequest{From: from.MarshalPB()}, &resp,
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

func ListMappingCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-mapping",
		Short: "List mapping address",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.AddressMapperListMappingResponse
			err := cli.StaticCallContractWithFlags(&flags,
				addressMapperContractName, "ListMapping",
				&address_mapper.AddressMapperListMappingRequest{}, &resp,
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const getBalanceCmdExample = `
# Get balance using a DAppChain address
loom staking balance 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

# Get balance using an Ethereum address
loom staking balance eth:0x0BE2BC95ea604a5ac4ECcE0F8570fe58bC9C320A
`

func GetBalanceCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "balance <owner hex address>",
		Short:   "Get balance on plasmachain",
		Example: getBalanceCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			var resp coin.BalanceOfResponse
			err = cli.StaticCallContractWithFlags(&flags, "coin", "BalanceOf", &coin.BalanceOfRequest{
				Owner: addr.MarshalPB(),
			}, &resp)
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

const getWithdrawalReceiptExample = `
# Get the withdrawal receipt using a Ethereum address
loom staking withdrawal-receipt eth:0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

Get the withdrawal receipt using a DappChain Address
loom staking withdrawal-receipt 0xCA08d2DB4563A64415bC16F17a0107A82DA622B7
`

func WithdrawalReceiptCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "withdrawal-receipt <owner hex address>",
		Short:   "Get the withdrawal receipt for an account",
		Example: getWithdrawalReceiptExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			var resp tgtypes.TransferGatewayWithdrawalReceiptResponse
			err = cli.StaticCallContractWithFlags(&flags,
				commands.LoomGatewayName, "WithdrawalReceipt",
				&tgtypes.TransferGatewayWithdrawalReceiptRequest{
					Owner: addr.MarshalPB(),
				},
				&resp,
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
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const checkAllDelegationsExample = `
# Display all delegations of a particular delegator using a Ethereum address
loom staking check-all-delegations eth:0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

Display all delegations of a particular delegator using a DappChain address
loom staking check-all-delegations 0xCA08d2DB4563A64415bC16F17a0107A82DA622B7
`

func CheckAllDelegationsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "check-all-delegations <delegator hex address>",
		Short:   "Display all of a particular delegator's delegations",
		Args:    cobra.MinimumNArgs(1),
		Example: checkAllDelegationsExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			var resp dposv2.CheckAllDelegationsResponse
			err = cli.StaticCallContractWithFlags(&flags, dPOSV2ContractName, "CheckAllDelegations", &dposv2.CheckAllDelegationsRequest{
				DelegatorAddress: addr.MarshalPB(),
			}, &resp)
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

const listCandidatesExample = `
# List all candidates
loom staking list-candidates
`

func ListCandidatesCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-candidates",
		Short:   "List the registered candidates",
		Example: listCandidatesExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListCandidateResponseV2
			err := cli.StaticCallContractWithFlags(
				&flags, dPOSV2ContractName, "ListCandidates", &dposv2.ListCandidateRequestV2{}, &resp,
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
