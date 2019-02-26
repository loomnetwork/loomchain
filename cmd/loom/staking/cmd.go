package staking

import (
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/builtin/commands"
	"github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/cli"
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

func ListDelegationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-delegations",
		Short: "list a candidate's delegations and delegation total",
		Args:  cobra.MinimumNArgs(1),
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

func GetMappingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-mapping",
		Short: "Get mapping address",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.AddressMapperGetMappingResponse
			from, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
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

// Utils

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
