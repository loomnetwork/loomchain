package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"

	"github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/types"
	"github.com/spf13/cobra"
)

const DPOSV2ContractName = "dposV2"

func UnregisterCandidateCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "unregister-candidate",
		Short: "Unregisters the candidate (only called if previously registered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "UnregisterCandidate",
				&dposv2.UnregisterCandidateRequestV2{}, nil,
			)
		},
	}
}

func GetDistributionsCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-distributions",
		Short: "Gets a list of all rewards for each address",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.GetDistributionsResponse
			err := cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "GetDistributions",
				&dposv2.GetDistributionsRequest{}, &resp,
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
}

func GetStateCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-dpos-state",
		Short: "Gets dpos state",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.GetStateResponse
			err := cli.StaticCallContractWithFlags(flags, DPOSV2ContractName, "GetState", &dposv2.GetStateRequest{}, &resp)
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
func ListValidatorsCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list-validators",
		Short: "List the current validators",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListValidatorsResponseV2
			err := cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "ListValidators",
				&dposv2.ListValidatorsRequestV2{}, &resp,
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
}

func ListCandidatesCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list-candidates",
		Short: "List the registered candidates",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListCandidateResponseV2
			err := cli.StaticCallContractWithFlags(flags, DPOSV2ContractName, "ListCandidates", &dposv2.ListCandidateRequestV2{}, &resp)
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

func ChangeFeeCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
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
				return errors.New("candidateFee is expressed in basis point (hundredths of a percent) and must be between 10000 (100%) and 0 (0%).")
			}
			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "ChangeFee", &dposv2.ChangeCandidateFeeRequest{
				Fee: candidateFee,
			}, nil)
		},
	}
}

func RegisterCandidateCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "register-candidate [public key] [validator fee (in basis points)] [locktime tier]",
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
			if len(args) == 3 {
				tier, err = strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return err
				}

				if tier > 3 {
					return errors.New("Tier value must be integer 0 - 4")
				}
			}

			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "RegisterCandidate2", &dposv2.RegisterCandidateRequestV2{
				PubKey:       pubKey,
				Fee:          candidateFee,
				Name:         candidateName,
				Description:  candidateDescription,
				Website:      candidateWebsite,
				LocktimeTier: tier,
			}, nil)
		},
	}
}

func UpdateCandidateInfoCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "update-candidate-info [name] [description] [website]",
		Short: "Update candidate information for a validator",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateName := args[0]
			candidateDescription := args[1]
			candidateWebsite := args[2]

			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "UpdateCandidateInfo", &dposv2.UpdateCandidateInfoRequest{
				Name:        candidateName,
				Description: candidateDescription,
				Website:     candidateWebsite,
			}, nil)
		},
	}
}

func DelegateCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delegate [validator address] [amount] [locktime tier]",
		Short: "delegate tokens to a validator",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			var req dposv2.DelegateRequestV2
			req.Amount = &types.BigUInt{Value: *amount}
			req.ValidatorAddress = addr.MarshalPB()

			if len(args) == 3 {
				tier, err := strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return err
				}

				if tier > 3 {
					return errors.New("Tier value must be integer 0 - 4")
				}

				req.LocktimeTier = tier
			}

			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "Delegate2", &req, nil)
		},
	}
}

func RedelegateCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "redelegate [new validator address] [former validator address] [amount]",
		Short: "Redelegate tokens from one validator to another",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			validatorAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			formerValidatorAddress, err := cli.ParseAddress(args[1])
			if err != nil {
				return err
			}

			var req dposv2.RedelegateRequestV2
			req.ValidatorAddress = validatorAddress.MarshalPB()
			req.FormerValidatorAddress = formerValidatorAddress.MarshalPB()

			if len(args) == 3 {
				amount, err := cli.ParseAmount(args[2])
				if err != nil {
					return err
				}
				req.Amount = &types.BigUInt{Value: *amount}
			}

			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "Redelegate", &req, nil)
		},
	}
}

func WhitelistCandidateCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "whitelist-candidate [candidate address] [amount] [lock time]",
		Short: "Whitelist candidate & credit candidate's self delegation without token deposit",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}
			locktime, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "WhitelistCandidate", &dposv2.WhitelistCandidateRequestV2{
				CandidateAddress: candidateAddress.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
				LockTime: locktime,
			}, nil)
		},
	}
}

func RemoveWhitelistedCandidateCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove-whitelisted-candidate [candidate address]",
		Short: "remove a candidate's whitelist entry",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			return cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "RemoveWhitelistedCandidate",
				&dposv2.RemoveWhitelistedCandidateRequestV2{
					CandidateAddress: candidateAddress.MarshalPB(),
				}, nil,
			)
		},
	}
}

func ChangeWhitelistLockTimeTierCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "change-whitelist-locktime-tier [candidate address] [amount]",
		Short: "Changes a whitelisted candidate's whitelist lock time tier",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			tier, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			if tier > 3 {
				return errors.New("Tier value must be integer 0 - 4")
			}

			return cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "ChangeWhitelistLockTimeTier",
				&dposv2.ChangeWhitelistLockTimeTierRequestV2{
					CandidateAddress: candidateAddress.MarshalPB(),
					LockTimeTier:     tier,
				}, nil,
			)
		},
	}
}

func ChangeWhitelistAmountCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "change-whitelist-amount [candidate address] [amount]",
		Short: "Changes a whitelisted candidate's whitelist amount",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			candidateAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}
			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "ChangeWhitelistAmount", &dposv2.ChangeWhitelistAmountRequestV2{
				CandidateAddress: candidateAddress.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}
}

func CheckDelegationCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check-delegation [validator address] [delegator address]",
		Short: "check delegation to a particular validator",
		Args:  cobra.MinimumNArgs(2),
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
			err = cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "CheckDelegation",
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
}

func UnbondCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "unbond [validator address] [amount]",
		Short: "De-allocate tokens from a validator",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			amount, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}
			return cli.CallContractWithFlags(flags, DPOSV2ContractName, "Unbond", &dposv2.UnbondRequestV2{
				ValidatorAddress: addr.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *amount,
				},
			}, nil)
		},
	}
}

func ClaimDistributionCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "claim-distribution [withdrawal address]",
		Short: "claim dpos distributions due to a validator or delegator",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			var resp dposv2.ClaimDistributionResponseV2
			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "ClaimDistribution",
				&dposv2.ClaimDistributionRequestV2{
					WithdrawalAddress: addr.MarshalPB(),
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
}

func CheckRewardsCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check-rewards",
		Short: "check rewards statistics",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.CheckRewardsResponse
			err := cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "CheckRewards",
				&dposv2.CheckRewardsRequest{}, &resp,
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
}

func CheckDistributionCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check-distribution [address]",
		Short: "check rewards distribution",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			var resp dposv2.CheckDistributionResponse
			err = cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "CheckDistribution",
				&dposv2.CheckDistributionRequest{
					Address: addr.MarshalPB(),
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
}

func TotalDelegationCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "total-delegation [delegator]",
		Short: "check how much a delegator has delegated in total (to all validators)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			var resp dposv2.TotalDelegationResponse
			err = cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "TotalDelegation",
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
}

func CheckAllDelegationsCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check-all-delegations [delegator]",
		Short: "display all of a particular delegator's delegations",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			var resp dposv2.CheckAllDelegationsResponse
			err = cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "CheckAllDelegations",
				&dposv2.CheckAllDelegationsRequest{DelegatorAddress: addr.MarshalPB()}, &resp,
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
}

func TimeUntilElectionCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "time-until-election",
		Short: "check how many seconds remain until the next election",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.TimeUntilElectionResponse
			err := cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "TimeUntilElection",
				&dposv2.TimeUntilElectionRequest{}, &resp,
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
}

func ListDelegationsCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list-delegations",
		Short: "list a candidate's delegations & delegation total",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}

			var resp dposv2.ListDelegationsResponse
			err = cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "ListDelegations",
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
}

func ListAllDelegationsCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list-all-delegations",
		Short: "display the results of calling list_delegations for all candidates",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv2.ListAllDelegationsResponse
			err := cli.StaticCallContractWithFlags(
				flags, DPOSV2ContractName, "ListAllDelegations",
				&dposv2.ListAllDelegationsRequest{}, &resp,
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
}

// Oracle Commands for setting parameters

func SetElectionCycleCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-election-cycle [election duration]",
		Short: "Set election cycle duration (in seconds)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			electionCycleDuration, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "SetElectionCycle",
				&dposv2.SetElectionCycleRequestV2{
					ElectionCycle: int64(electionCycleDuration),
				}, nil,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func SetValidatorCountCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-validator-count [validator count]",
		Short: "Set maximum number of validators",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			validatorCount, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "SetValidatorCount2",
				&dposv2.SetValidatorCountRequestV2{
					ValidatorCount: int64(validatorCount),
				}, nil,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func SetMaxYearlyRewardCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-max-yearly-reward [max yearly rewward amount]",
		Short: "Set maximum yearly reward",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxYearlyReward, err := cli.ParseAmount(args[0])
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "SetMaxYearlyReward",
				&dposv2.SetMaxYearlyRewardRequestV2{
					MaxYearlyReward: &types.BigUInt{
						Value: *maxYearlyReward,
					},
				}, nil,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func SetRegistrationRequirementCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-registration-requirement [registration_requirement]",
		Short: "Set minimum self-delegation required of a new Candidate",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registrationRequirement, err := cli.ParseAmount(args[0])
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "SetRegistrationRequirement",
				&dposv2.SetRegistrationRequirementRequestV2{
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
}

func SetOracleAddressCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-oracle-address [oracle address]",
		Short: "Set oracle address",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			oracleAddress, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "SetOracleAddress",
				&dposv2.SetOracleAddressRequestV2{OracleAddress: oracleAddress.MarshalPB()}, nil,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func SetSlashingPercentagesCmdV2(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-slashing-percentages [crash fault slashing percentage] [byzantine fault slashing percentage",
		Short: "Set crash and byzantine fualt slashing percentages expressed in basis points",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			registrationRequirement, err := cli.ParseAmount(args[1])
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				flags, DPOSV2ContractName, "SetRegistrationRequirement",
				&dposv2.SetRegistrationRequirementRequestV2{
					RegistrationRequirement: &types.BigUInt{
						Value: *registrationRequirement,
					},
				}, nil,
			)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func NewDPOSV2Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dposV2 <command>",
		Short: "Methods available in dposv2 contract",
	}
	var flags cli.ContractCallFlags
	AddContractCallFlags(cmd.PersistentFlags(), &flags)
	registercmd := RegisterCandidateCmdV2(&flags)
	registercmd.Flags().StringVarP(&candidateName, "name", "", "", "candidate name")
	registercmd.Flags().StringVarP(&candidateDescription, "description", "", "", "candidate description")
	registercmd.Flags().StringVarP(&candidateWebsite, "website", "", "", "candidate website")

	cmd.AddCommand(
		registercmd,
		ListCandidatesCmdV2(&flags),
		ListValidatorsCmdV2(&flags),
		ListDelegationsCmd(&flags),
		ListAllDelegationsCmd(&flags),
		UnregisterCandidateCmdV2(&flags),
		UpdateCandidateInfoCmd(&flags),
		DelegateCmdV2(&flags),
		RedelegateCmdV2(&flags),
		WhitelistCandidateCmdV2(&flags),
		RemoveWhitelistedCandidateCmdV2(&flags),
		ChangeWhitelistAmountCmdV2(&flags),
		ChangeWhitelistLockTimeTierCmdV2(&flags),
		CheckDelegationCmdV2(&flags),
		CheckAllDelegationsCmd(&flags),
		CheckDistributionCmd(&flags),
		CheckRewardsCmd(&flags),
		UnbondCmdV2(&flags),
		ClaimDistributionCmdV2(&flags),
		SetElectionCycleCmdV2(&flags),
		SetValidatorCountCmdV2(&flags),
		SetMaxYearlyRewardCmdV2(&flags),
		SetRegistrationRequirementCmdV2(&flags),
		SetOracleAddressCmdV2(&flags),
		SetSlashingPercentagesCmdV2(&flags),
		ChangeFeeCmd(&flags),
		TimeUntilElectionCmd(&flags),
		TotalDelegationCmd(&flags),
		GetStateCmd(&flags),
		GetDistributionsCmd(&flags),
	)
	return cmd

}
