package staking

import (
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/commands"
	"github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/gateway"
	"github.com/spf13/cobra"
)

func NewStakingCommand() *cobra.Command {
	cmd := cli.ContractCallCommand("staking")
	cmd.Use = "staking"
	cmd.Short = "Run staking commands"
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
	return &cobra.Command{
		Use:   "list-all-delegations",
		Short: "display the all delegations",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListAllDelegationsResponse
			err := cli.StaticCallContract(commands.DPOSV2ContractName, "ListAllDelegations", &dposv2.ListAllDelegationsRequest{}, &resp)
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

const listDelegationsCmdExample = `
loom staking list-delegations 0x0ca3d6bf201ce53c7ddc3cb397ae33a68ed4a328
`

func ListDelegationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list-delegations <validator hex address>",
		Short:   "List a validator's delegations and delegation total",
		Example: listDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0])
			if err != nil {
				return err
			}

			var resp dposv2.ListDelegationsResponse
			err = cli.StaticCallContract(commands.DPOSV2ContractName, "ListDelegations", &dposv2.ListDelegationsRequest{Candidate: addr.MarshalPB()}, &resp)
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

func ListValidatorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-validators",
		Short: "List the current validators",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListValidatorsResponseV2
			err := cli.StaticCallContract(commands.DPOSV2ContractName, "ListValidators", &dposv2.ListValidatorsRequestV2{}, &resp)
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

const totalDelegationCmdExample = `
# Get total delegations using a DAppChain address
loom staking total-delegation 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

# Get total delegations using an Ethereum address
loom staking total-delegation eth:0x751481F4db7240f4d5ab5d8c3A5F6F099C824863
`

func TotalDelegationCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "total-delegation (delegator address)",
		Short:   "Display total staking amount that has delegated to all validators",
		Example: totalDelegationCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0])
			if err != nil {
				return err
			}

			if addr.ChainID == "eth" {
				var resp address_mapper.AddressMapperGetMappingResponse
				var req = address_mapper.AddressMapperGetMappingRequest{
					From: addr.MarshalPB(),
				}
				err = cli.StaticCallContract(commands.AddressMapperContractName, "GetMapping", &req, &resp)
				if err != nil {
					return err
				}
				addr = loom.UnmarshalAddressPB(resp.To)
			}

			var resp dposv2.TotalDelegationResponse
			err = cli.StaticCallContract(commands.DPOSV2ContractName, "TotalDelegation", &dposv2.TotalDelegationRequest{DelegatorAddress: addr.MarshalPB()}, &resp)
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

const checkDelegationsCmdExample = `
loom staking check-delegation 0x0ca3d6bf201ce53c7ddc3cb397ae33a68ed4a328 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863
`

func CheckDelegationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "check-delegation (validator address) (delegator address)",
		Short:   "Check delegation to a particular validator",
		Example: checkDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.CheckDelegationResponseV2
			validatorAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			delegatorAddress, err := cli.ParseAddress(args[1])
			if err != nil {
				return err
			}
			err = cli.StaticCallContract(commands.DPOSV2ContractName, "CheckDelegation", &dposv2.CheckDelegationRequestV2{ValidatorAddress: validatorAddress.MarshalPB(), DelegatorAddress: delegatorAddress.MarshalPB()}, &resp)
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

const getMappingCmdExample = `
# Get mapping address using a DAppChain address
loom staking get-mapping 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

# Get mapping using an Ethereum address
loom staking get-mapping eth:0x0BE2BC95ea604a5ac4ECcE0F8570fe58bC9C320A
`

func GetMappingCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get-mapping",
		Short:   "Get mapping address",
		Example: getMappingCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.AddressMapperGetMappingResponse
			from, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			if from.ChainID == "eth" {
				cli.TxFlags.ChainID = "eth"
			}

			err = cli.StaticCallContract(commands.AddressMapperContractName, "GetMapping", &address_mapper.AddressMapperGetMappingRequest{
				From: from.MarshalPB(),
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
}

func ListMappingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-mapping",
		Short: "List mapping address",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.AddressMapperListMappingResponse
			err := cli.StaticCallContract(commands.AddressMapperContractName, "ListMapping", &address_mapper.AddressMapperListMappingRequest{}, &resp)
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

const getBalanceCmdExample = `
# Get balance using a DAppChain address
loom staking balance 0x751481F4db7240f4d5ab5d8c3A5F6F099C824863

# Get balance using an Ethereum address
loom staking balance eth:0x0BE2BC95ea604a5ac4ECcE0F8570fe58bC9C320A
`

func GetBalanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "balance <owner hex address>",
		Short:   "Get balance on plasmachain",
		Example: checkDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0])
			if err != nil {
				return err
			}

			if addr.ChainID == "eth" {
				var resp address_mapper.AddressMapperGetMappingResponse
				var req = address_mapper.AddressMapperGetMappingRequest{
					From: addr.MarshalPB(),
				}
				err = cli.StaticCallContract(commands.AddressMapperContractName, "GetMapping", &req, &resp)
				if err != nil {
					return err
				}
				addr = loom.UnmarshalAddressPB(resp.To)
			}

			var resp coin.BalanceOfResponse
			err = cli.StaticCallContract(commands.CoinContractName, "BalanceOf", &coin.BalanceOfRequest{
				Owner: addr.MarshalPB(),
			}, &resp)
			out, err := formatJSON(&resp)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}

const getWithdrawalReceiptExample = `
# Get withdrawal receipt using a Ethereum address
loom staking withdrawal-receipt eth:0x751481F4db7240f4d5ab5d8c3A5F6F099C824863
`

func WithdrawalReceiptCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "withdrawal-receipt <owner hex address>",
		Short:   "Get withdrawal receipt of an account on plasmachain",
		Example: getWithdrawalReceiptExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0])
			if err != nil {
				return err
			}

			if addr.ChainID == "eth" {
				var resp address_mapper.AddressMapperGetMappingResponse
				var req = address_mapper.AddressMapperGetMappingRequest{
					From: addr.MarshalPB(),
				}
				err = cli.StaticCallContract(commands.AddressMapperContractName, "GetMapping", &req, &resp)
				if err != nil {
					return err
				}
				addr = loom.UnmarshalAddressPB(resp.To)
			}

			var resp gateway.WithdrawalReceiptResponse
			err = cli.StaticCallContract(commands.LoomGatewayName, "WithdrawalReceipt", &gateway.WithdrawalReceiptRequest{
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
}

const checkAllDelegationsExample = `
# Check all delegation using a Ethereum address
loom staking check-all-delegations eth:0x751481F4db7240f4d5ab5d8c3A5F6F099C824863
`

func CheckAllDelegationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "check-all-delegations <delegator hex address>",
		Short:   "Get all delegations of this delegator on plasmachain",
		Args:    cobra.MinimumNArgs(1),
		Example: checkAllDelegationsExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAddress(args[0])
			if err != nil {
				return err
			}

			if addr.ChainID == "eth" {
				var resp address_mapper.AddressMapperGetMappingResponse
				var req = address_mapper.AddressMapperGetMappingRequest{
					From: addr.MarshalPB(),
				}
				err = cli.StaticCallContract(commands.AddressMapperContractName, "GetMapping", &req, &resp)
				if err != nil {
					return err
				}
				addr = loom.UnmarshalAddressPB(resp.To)
			}

			var resp dposv2.CheckAllDelegationsResponse
			err = cli.StaticCallContract(commands.DPOSV2ContractName, "CheckAllDelegations", &dposv2.CheckAllDelegationsRequest{
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
}

const listCandidatesExample = `
# List all candidates 
loom staking list candidates
`

func ListCandidatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list-candidates",
		Short:   "Get all candidates on plasmachain",
		Example: listCandidatesExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListCandidateResponseV2
			err := cli.StaticCallContract(commands.DPOSV2ContractName, "ListCandidates", &dposv2.ListCandidateRequestV2{}, &resp)
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
