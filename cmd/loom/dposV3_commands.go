package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/types"
)

const DPOSV3ContractName = "dposV3"

var (
	candidateName        string
	candidateDescription string
	candidateWebsite     string
)

func UnregisterCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "unregister-candidate",
		Short: "Unregisters the candidate (only called if previously registered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "UnregisterCandidate", &dposv3.UnregisterCandidateRequest{}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func UnjailValidatorCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags

	cmd := &cobra.Command{
		Use:   "unjail-validator",
		Short: "Unjail a validator",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var validator *types.Address
			if len(args) == 1 {
				addr, err := cli.ParseAddress(args[0], flags.ChainID)
				if err != nil {
					return err
				}
				validator = addr.MarshalPB()
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "Unjail", &dposv3.UnjailRequest{
					Validator: validator,
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func GetStateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "get-dpos-state",
		Short: "Gets dpos state",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.GetStateResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "GetState", &dposv3.GetStateRequest{}, &resp,
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

func ListValidatorsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-validators",
		Short: "List the current validators",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.ListValidatorsResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListValidators", &dposv3.ListValidatorsRequest{}, &resp,
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

func ListCandidatesCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-candidates",
		Short: "List the registered candidates",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.ListCandidatesResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListCandidates", &dposv3.ListCandidatesRequest{}, &resp,
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

func ChangeFeeCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "change-fee [new validator fee (in basis points)]",
		Short: "Changes a validator's fee after (with a 2 election delay)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateFee, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			if candidateFee > 10000 {
				// nolint:lll
				return errors.New("candidateFee is expressed in basis points (hundredths of a percent) and must be between 10000 (100%) and 0 (0%).")
			}
			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "ChangeFee", &dposv3.ChangeCandidateFeeRequest{
					Fee: candidateFee,
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func RegisterCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		// nolint:lll
		Use: "register-candidate [public key] [validator fee (" +
			"in basis points)] [locktime tier] [maximum referral percentage]",
		Short: "Register a candidate for validator",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey, err := base64.StdEncoding.DecodeString(args[0])
			if err != nil {
				return err
			}
			candidateFee, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			if candidateFee > 10000 {
				// nolint:lll
				return errors.New("candidateFee is expressed in basis point (hundredths of a percent) and must be between 10000 (100%) and 0 (0%).")
			}

			tier := uint64(0)
			if len(args) >= 3 {
				tier, err = strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return err
				}

				if tier > 3 {
					return errors.New("Tier value must be integer 0 - 3")
				}
			}

			maxReferralPercentage := uint64(0)
			if len(args) >= 4 {
				maxReferralPercentage, err = strconv.ParseUint(args[3], 10, 64)
				if err != nil {
					return err
				}
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "RegisterCandidate",
				&dposv3.RegisterCandidateRequest{
					PubKey:                pubKey,
					Fee:                   candidateFee,
					Name:                  candidateName,
					Description:           candidateDescription,
					Website:               candidateWebsite,
					LocktimeTier:          tier,
					MaxReferralPercentage: maxReferralPercentage,
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func UpdateCandidateInfoCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "update-candidate-info [name] [description] [website] [maximum referral percentage]",
		Short: "Update candidate information for a validator",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateName := args[0]
			candidateDescription := args[1]
			candidateWebsite := args[2]
			maxReferralPercentage := uint64(0)
			if len(args) >= 4 {
				percentage, err := strconv.ParseUint(args[3], 10, 64)
				if err != nil {
					return err
				}
				if percentage > 10000 {
					// nolint:lll
					return errors.New("maxReferralFee is expressed in basis points (hundredths of a percent) and must be between 10000 (100%) and 0 (0%).")
				}
				maxReferralPercentage = percentage
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "UpdateCandidateInfo", &dposv3.UpdateCandidateInfoRequest{
					Name:                  candidateName,
					Description:           candidateDescription,
					Website:               candidateWebsite,
					MaxReferralPercentage: maxReferralPercentage,
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func DelegateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "delegate [validator address] [amount] [locktime tier] [referrer]",
		Short: "delegate tokens to a validator",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			var req dposv3.DelegateRequest
			req.Amount = &types.BigUInt{Value: *amount}
			req.ValidatorAddress = addr.MarshalPB()

			if len(args) >= 3 {
				tier, err := strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return err
				}

				if tier > 3 {
					return errors.New("Tier value must be integer 0 - 3")
				}

				req.LocktimeTier = tier
			}

			if len(args) >= 4 {
				req.Referrer = args[3]
			}

			return cli.CallContractWithFlags(&flags, DPOSV3ContractName, "Delegate", &req, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func RedelegateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "redelegate [new validator address] [former validator address] [index] [amount] [referrer]",
		Short: "Redelegate tokens from one validator to another",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			validatorAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			formerValidatorAddress, err := cli.ParseAddress(args[1], flags.ChainID)
			if err != nil {
				return err
			}

			index, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			var req dposv3.RedelegateRequest
			req.ValidatorAddress = validatorAddress.MarshalPB()
			req.FormerValidatorAddress = formerValidatorAddress.MarshalPB()
			req.Index = index

			if len(args) >= 4 {
				amount, err := cli.ParseAmount(args[3])
				if err != nil {
					return err
				}
				req.Amount = &types.BigUInt{Value: *amount}
			}

			if len(args) >= 5 {
				req.Referrer = args[4]
			}

			return cli.CallContractWithFlags(&flags, DPOSV3ContractName, "Redelegate", &req, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func WhitelistCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "whitelist-candidate [candidate address] [amount] [locktime tier]",
		Short: "Whitelist candidate & credit candidate's self delegation without token deposit",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			tier := uint64(0)
			if len(args) >= 3 {
				tier, err = strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return err
				}

				if tier > 3 {
					return errors.New("Tier value must be integer 0 - 3")
				}
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "WhitelistCandidate",
				&dposv3.WhitelistCandidateRequest{
					CandidateAddress: candidateAddress.MarshalPB(),
					Amount: &types.BigUInt{
						Value: *amount,
					},
					LocktimeTier: dposv3.LocktimeTier(tier),
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func RemoveWhitelistedCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "remove-whitelisted-candidate [candidate address]",
		Short: "remove a candidate's whitelist entry",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "RemoveWhitelistedCandidate",
				&dposv3.RemoveWhitelistedCandidateRequest{
					CandidateAddress: candidateAddress.MarshalPB(),
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func ChangeWhitelistInfoCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "change-whitelist-info [candidate address] [amount] [locktime tier]",
		Short: "Changes a whitelisted candidate's whitelist amount and tier",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			tier := uint64(0)
			if len(args) >= 3 {
				tier, err = strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return err
				}

				if tier > 3 {
					return errors.New("Tier value must be integer 0 - 3")
				}
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "ChangeWhitelistInfo",
				&dposv3.ChangeWhitelistInfoRequest{
					CandidateAddress: candidateAddress.MarshalPB(),
					Amount: &types.BigUInt{
						Value: *amount,
					},
					LocktimeTier: dposv3.LocktimeTier(tier),
				}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func CheckDelegationCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "check-delegation [validator address] [delegator address]",
		Short: "check delegation to a particular validator",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.CheckDelegationResponse
			validatorAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			delegatorAddress, err := cli.ResolveAccountAddress(args[1], &flags)
			if err != nil {
				return err
			}
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "CheckDelegation",
				&dposv3.CheckDelegationRequest{
					ValidatorAddress: validatorAddress.MarshalPB(),
					DelegatorAddress: delegatorAddress.MarshalPB()},
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

func DowntimeRecordCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "downtime-record [validator address]",
		Short: "check a validator's downtime record",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var validatorAddress *types.Address
			if len(args) > 0 {
				address, err := cli.ParseAddress(args[0], flags.ChainID)
				if err != nil {
					return err
				}
				validatorAddress = address.MarshalPB()
			}

			var resp dposv3.DowntimeRecordResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "DowntimeRecord",
				&dposv3.DowntimeRecordRequest{
					Validator: validatorAddress,
				}, &resp,
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

			type mapper struct {
				Address        string
				Name           string
				DownTimeRecord *dposv3.DowntimeRecord
				Jailed         bool
			}
			var nameList []mapper

			for _, d := range resp.DowntimeRecords {
				for _, c := range respDPOS.Candidates {
					if d.Validator.Local.Compare(c.Candidate.Address.Local) == 0 {
						a := mapper{
							Address:        loom.UnmarshalAddressPB(d.GetValidator()).Local.String(),
							Name:           c.Candidate.GetName(),
							DownTimeRecord: d,
							Jailed:         c.Statistic.Jailed,
						}
						nameList = append(nameList, a)
						break
					}
				}
			}

			sort.Slice(nameList[:], func(i, j int) bool {
				return nameList[i].Name < nameList[j].Name
			})

			type maxLength struct {
				Name    int
				Address int
				Period  int
				Jailed  int
			}
			ml := maxLength{Name: 40, Address: 42, Period: 5, Jailed: 6}
			fmt.Printf(
				"%-*s | %-*s | %-*s | %*s | %*s | %*s | %*s |\n", ml.Name, "name", ml.Address, "address",
				ml.Jailed, "jailed", ml.Period, "P", ml.Period, "P-1", ml.Period, "P-2", ml.Period, "P-3")
			fmt.Printf(
				strings.Repeat("-", ml.Name+ml.Address+ml.Jailed+(4*ml.Period)+19) + "\n")
			for i := range nameList {
				fmt.Printf(
					"%-*s | %-*s | %*v | %*d | %*d | %*d | %*d |\n",
					ml.Name, nameList[i].Name,
					ml.Address, nameList[i].Address,
					ml.Jailed, nameList[i].Jailed,
					ml.Period, nameList[i].DownTimeRecord.Periods[0],
					ml.Period, nameList[i].DownTimeRecord.Periods[1],
					ml.Period, nameList[i].DownTimeRecord.Periods[2],
					ml.Period, nameList[i].DownTimeRecord.Periods[3])
			}
			fmt.Println("PeriodLength : ", resp.PeriodLength)
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

func UnbondCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "unbond [validator address] [amount] [index]",
		Short: "De-allocate tokens from a validator",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			index, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(&flags, DPOSV3ContractName, "Unbond", &dposv3.UnbondRequest{
				ValidatorAddress: addr.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
				Index: index,
			}, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func ClaimDelegatorRewardsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "claim-delegator-rewards",
		Short: "claim pending delegation rewards",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.ClaimDelegatorRewardsResponse
			err := cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "ClaimRewardsFromAllValidators",
				&dposv3.ClaimDelegatorRewardsRequest{}, &resp,
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

func CheckDelegatorRewardsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "check-delegator-rewards <address>",
		Short: "check rewards for the specified delegator",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			var resp dposv3.CheckDelegatorRewardsResponse
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "CheckRewardsFromAllValidators",
				&dposv3.CheckDelegatorRewardsRequest{
					Delegator: address.MarshalPB(),
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

func CheckRewardsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "check-rewards",
		Short: "check rewards statistics",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.CheckRewardsResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "CheckRewards", &dposv3.CheckRewardsRequest{}, &resp,
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

func CheckAllDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "check-all-delegations [delegator]",
		Short: "display all of a particular delegator's delegations",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}

			var resp dposv3.CheckAllDelegationsResponse
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "CheckAllDelegations",
				&dposv3.CheckAllDelegationsRequest{DelegatorAddress: addr.MarshalPB()}, &resp,
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

func TimeUntilElectionCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "time-until-election",
		Short: "check how many seconds remain until the next election",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.TimeUntilElectionResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "TimeUntilElection",
				&dposv3.TimeUntilElectionRequest{}, &resp,
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

func ListDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-delegations <candidate address>",
		Short: "list a candidate's delegations & delegation total",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			var resp dposv3.ListDelegationsResponse
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListDelegations",
				&dposv3.ListDelegationsRequest{Candidate: addr.MarshalPB()}, &resp,
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

func ListAllDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-all-delegations",
		Short: "display the results of calling list_delegations for all candidates",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.ListAllDelegationsResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListAllDelegations",
				&dposv3.ListAllDelegationsRequest{}, &resp,
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

// Oracle Commands for setting parameters

func RegisterReferrerCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "register-referrer [name] [address]",
		Short: "Register a referrer wallet's name and address",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			address, err := cli.ParseAddress(args[1], flags.ChainID)
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "RegisterReferrer", &dposv3.RegisterReferrerRequest{
					Name:    name,
					Address: address.MarshalPB(),
				}, nil)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetElectionCycleCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-election-cycle [election duration]",
		Short: "Set election cycle duration (in seconds)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			electionCycleDuration, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetElectionCycle", &dposv3.SetElectionCycleRequest{
					ElectionCycle: int64(electionCycleDuration),
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetDowntimePeriodCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-downtime-period [downtime period]",
		Short: "Set downtime period duration (in blocks)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			downtimePeriod, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetDowntimePeriod", &dposv3.SetDowntimePeriodRequest{
					DowntimePeriod: downtimePeriod,
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetValidatorCountCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-validator-count [validator count]",
		Short: "Set maximum number of validators",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			validatorCount, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetValidatorCount", &dposv3.SetValidatorCountRequest{
					ValidatorCount: int64(validatorCount),
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetMaxYearlyRewardCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-max-yearly-reward [max yearly rewward amount]",
		Short: "Set maximum yearly reward",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxYearlyReward, err := cli.ParseAmount(args[0])
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetMaxYearlyReward", &dposv3.SetMaxYearlyRewardRequest{
					MaxYearlyReward: &types.BigUInt{
						Value: *maxYearlyReward,
					},
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetRegistrationRequirementCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-registration-requirement [registration_requirement]",
		Short: "Set minimum self-delegation required of a new Candidate",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registrationRequirement, err := cli.ParseAmount(args[0])
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetRegistrationRequirement", &dposv3.SetRegistrationRequirementRequest{
					RegistrationRequirement: &types.BigUInt{
						Value: *registrationRequirement,
					},
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetOracleAddressCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-oracle-address [oracle address]",
		Short: "Set oracle address",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			oracleAddress, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}
			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetOracleAddress",
				&dposv3.SetOracleAddressRequest{OracleAddress: oracleAddress.MarshalPB()}, nil,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetSlashingPercentagesCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-slashing-percentages [crash fault slashing percentage] [byzantine fault slashing percentage",
		Short: "Set crash and byzantine fualt slashing percentages expressed in basis points",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			crashFaultSlashingPercentage, err := cli.ParseAmount(args[0])
			if err != nil {
				return err
			}
			byzantineFaultSlashingPercentage, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetSlashingPercentages", &dposv3.SetSlashingPercentagesRequest{
					CrashSlashingPercentage: &types.BigUInt{
						Value: *crashFaultSlashingPercentage,
					},
					ByzantineSlashingPercentage: &types.BigUInt{
						Value: *byzantineFaultSlashingPercentage,
					},
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func SetMinCandidateFeeCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-min-candidate-fee [min candidate fee]",
		Short: "Set minimum candidate fee",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			minCandidateFee, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			if minCandidateFee > 10000 {
				// nolint:lll
				return errors.New("minCandidateFee is expressed in basis point (hundredths of a percent) and must be between 10000 (100%) and 0 (0%).")
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetMinCandidateFee", &dposv3.SetMinCandidateFeeRequest{
					MinCandidateFee: minCandidateFee,
				}, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func NewDPOSV3Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dpos3 <command>",
		Short: "Methods available in dposv3 contract",
	}
	registercmd := RegisterCandidateCmdV3()
	registercmd.Flags().StringVarP(&candidateName, "name", "", "", "candidate name")
	registercmd.Flags().StringVarP(&candidateDescription, "description", "", "", "candidate description")
	registercmd.Flags().StringVarP(&candidateWebsite, "website", "", "", "candidate website")
	cmd.AddCommand(
		registercmd,
		ListCandidatesCmdV3(),
		ListValidatorsCmdV3(),
		ListDelegationsCmdV3(),
		ListAllDelegationsCmdV3(),
		UnregisterCandidateCmdV3(),
		UpdateCandidateInfoCmdV3(),
		DelegateCmdV3(),
		RedelegateCmdV3(),
		WhitelistCandidateCmdV3(),
		RemoveWhitelistedCandidateCmdV3(),
		ChangeWhitelistInfoCmdV3(),
		CheckDelegatorRewardsCmdV3(),
		ClaimDelegatorRewardsCmdV3(),
		CheckDelegationCmdV3(),
		CheckAllDelegationsCmdV3(),
		CheckRewardsCmdV3(),
		DowntimeRecordCmdV3(),
		UnbondCmdV3(),
		RegisterReferrerCmdV3(),
		SetDowntimePeriodCmdV3(),
		SetElectionCycleCmdV3(),
		SetValidatorCountCmdV3(),
		SetMaxYearlyRewardCmdV3(),
		SetRegistrationRequirementCmdV3(),
		SetOracleAddressCmdV3(),
		SetSlashingPercentagesCmdV3(),
		ChangeFeeCmdV3(),
		TimeUntilElectionCmdV3(),
		GetStateCmdV3(),
		SetMinCandidateFeeCmdV3(),
		UnjailValidatorCmdV3(),
	)
	return cmd
}
