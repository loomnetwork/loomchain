package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const DPOSV3ContractName = "dposV3"

var (
	candidateName        string
	candidateDescription string
	candidateWebsite     string
)

const unregisterCandidateCmdExample = ` 
loom dpos3 unregister-candidate --key path/to/private_key
loom dpos3 unregister-candidate <candidate address> --key path/to/oracle_private_key
`

func UnregisterCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "unregister-candidate",
		Short:   "Unregister a previously registered candidate",
		Example: unregisterCandidateCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var candidateAddrPB *types.Address
			if len(args) == 1 {
				addr, err := cli.ParseAddress(args[0], flags.ChainID)
				if err != nil {
					return err
				}
				candidateAddrPB = addr.MarshalPB()
			}
			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "UnregisterCandidate",
				&dposv3.UnregisterCandidateRequest{Candidate: candidateAddrPB}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const unjailValidatorCmdExample = `
loom dpos3 unjail-validator --key path/to/private_key
`

func UnjailValidatorCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags

	cmd := &cobra.Command{
		Use:     "unjail-validator",
		Short:   "Unjail a validator",
		Example: unjailValidatorCmdExample,
		Args:    cobra.RangeArgs(0, 1),
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

const getStateCmdExample = `
loom dpos3 get-dpos-state
`

func GetStateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "get-dpos-state",
		Short:   "Gets dpos state",
		Example: getStateCmdExample,
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

const listValidatorsCmdExample = `
loom dpos3 list-validators
`

func ListValidatorsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-validators",
		Short:   "List the current validators",
		Example: listValidatorsCmdExample,
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

const listCandidateCmdExample = `
loom dpos3 list-candidates
`

func ListCandidatesCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-candidates",
		Short:   "List the registered candidates",
		Example: listCandidateCmdExample,
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

const listReferrersCmdExample = `
loom dpos3 list-referrers 
`

func ListReferrersCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-referrers",
		Short:   "List all registered referrers",
		Example: listReferrersCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp dposv3.ListReferrersResponse
			err := cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListReferrers", &dposv3.ListReferrersRequest{}, &resp,
			)
			if err != nil {
				return err
			}
			type maxLength struct {
				Name    int
				Address int
			}
			ml := maxLength{Name: 20, Address: 50}

			for _, r := range resp.Referrers {
				if ml.Name < len(r.Name) {
					ml.Name = len(r.Name)
				}
			}

			fmt.Printf("%-*s | %-*s \n", ml.Name, "referrer name", ml.Address, "address")
			fmt.Printf(strings.Repeat("-", ml.Name+ml.Address+4) + "\n")
			for _, r := range resp.Referrers {
				fmt.Printf(
					"%-*s | %-*s "+"\n",
					ml.Name, r.Name, ml.Address, loom.UnmarshalAddressPB(r.GetReferrerAddress()).String(),
				)
			}

			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
}

const changeFeeCmdExample = `
loom dpos3 change-fee 2000 --k path/to/private_key
`

func ChangeFeeCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "change-fee [new validator fee (in basis points)]",
		Short:   "Changes a validator's fee after (with a 2 election delay)",
		Example: changeFeeCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const registerCandidateCmdExample = `
loom dpos3 register-candidate 0x7262d4c97c7B93937E4810D289b7320e9dA82857 100 3 --name candidate_name
`

func RegisterCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		// nolint:lll
		Use: "register-candidate [public key] [validator fee (" +
			"in basis points)] [locktime tier] [maximum referral percentage]",
		Short:   "Register a candidate for validator",
		Example: registerCandidateCmdExample,
		Args:    cobra.MinimumNArgs(2),
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

const updateCandidateCmdExample = `
loom dpos3 update-candidate-info candidate_name candidate_description candidate.com 1000 --key path/to/private_key
`

func UpdateCandidateInfoCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "update-candidate-info [name] [description] [website] [maximum referral percentage]",
		Short:   "Update candidate information for a validator",
		Example: updateCandidateCmdExample,
		Args:    cobra.MinimumNArgs(3),
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

const delegateCmdExample = `
loom dpos3 delegate 0x7262d4c97c7B93937E4810D289b7320e9dA82857 100 0 referrer_name
`

func DelegateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "delegate [validator address] [amount] [locktime tier] [referrer]",
		Short:   "delegate tokens to a validator",
		Example: delegateCmdExample,
		Args:    cobra.MinimumNArgs(2),
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

const redelegateCmdExample = `
loom dpos3 redelegate 0x7262d4c97c7B93937E4810D289b7320e9dA82857 0x62666100f8988238d81831dc543D098572F283A1 1 -k path/to/private_key
`

func RedelegateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "redelegate [new validator address] [former validator address] [index] [amount] [referrer]",
		Short:   "Redelegate tokens from one validator to another",
		Example: redelegateCmdExample,
		Args:    cobra.MinimumNArgs(3),
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

const whiteListCandidateCmdExample = `
loom dpos3 whitelist-candidate 0x7262d4c97c7B93937E4810D289b7320e9dA82857 1250000 0 -k path/to/private_key
`

func WhitelistCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "whitelist-candidate [candidate address] [amount] [locktime tier]",
		Short:   "Whitelist candidate & credit candidate's self delegation without token deposit",
		Example: whiteListCandidateCmdExample,
		Args:    cobra.MinimumNArgs(3),
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

const removeWhitelistCandidateCmdExample = `
loom dpos3 remove-whitelisted-candidate 0x7262d4c97c7B93937E4810D289b7320e9dA82857 -k path/to/private_key
`

func RemoveWhitelistedCandidateCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "remove-whitelisted-candidate [candidate address]",
		Short:   "remove a candidate's whitelist entry",
		Example: removeWhitelistCandidateCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const changeWhitelistInfoCmdExample = `
loom dpos3 change-whitelist-info 0x7262d4c97c7B93937E4810D289b7320e9dA82857 130000 0 --key path\to\private_key
`

func ChangeWhitelistInfoCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "change-whitelist-info [candidate address] [amount] [locktime tier]",
		Short:   "Changes a whitelisted candidate's whitelist amount and tier",
		Example: changeWhitelistInfoCmdExample,
		Args:    cobra.MinimumNArgs(2),
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

const checkDelegationCmdExample = `
loom dpos3 check-delegation 0x7262d4c97c7B93937E4810D289b7320e9dA82857 0x62666100f8988238d81831dc543D098572F283A1
`

func CheckDelegationCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "check-delegation [validator address] [delegator address]",
		Short:   "check delegation to a particular validator",
		Example: checkDelegationCmdExample,
		Args:    cobra.MinimumNArgs(2),
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

const downtimeRecordExample = `
loom dpos3 downtime-record 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func DowntimeRecordCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "downtime-record [validator address]",
		Short:   "check a validator's downtime record",
		Example: downtimeRecordExample,
		Args:    cobra.RangeArgs(0, 1),
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

const unbondCmdExample = `
loom dpos3 unbond 0x7262d4c97c7B93937E4810D289b7320e9dA82857 10 0 --key path/to/private_key
`

func UnbondCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "unbond [validator address] [amount] [index]",
		Short:   "De-allocate tokens from a validator",
		Example: unbondCmdExample,
		Args:    cobra.MinimumNArgs(3),
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

const unbondAllCmdExample = `
loom dpos3 unbond-all 0x7262d4c97c7B93937E4810D289b7320e9dA82857 -k path/to/private_key
`

func UnbondAllDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "unbond-all [validator address]",
		Short:   "unbond all delegations of a target validator",
		Example: unbondAllCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cli.ResolveAccountAddress(args[0], &flags)
			if err != nil {
				return err
			}
			return cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "UnbondAll",
				&dposv3.UnbondAllRequest{ValidatorAddress: addr.MarshalPB()}, nil,
			)
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const redelegateAllCmdExample = `
loom dpos3 redelegate-all 0x7262d4c97c7B93937E4810D289b7320e9dA82857 -k path/to/private_key
`

func RedelegateAllDelegationsCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "redelegate-all [validator address]",
		Short:   "Randomly redelegate all delegations from one validator to other validators",
		Example: redelegateAllCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcValidatorAddr, err := cli.ParseAddress(args[0], flags.ChainID)
			if err != nil {
				return err
			}

			// check the key matches the oracle
			signer, err := cli.GetSigner(flags.PrivFile, flags.HsmConfigFile, flags.Algo)
			if err != nil {
				return err
			}
			signerAddr := loom.LocalAddressFromPublicKey(signer.PublicKey())

			var stateResp dposv3.GetStateResponse
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "GetState", &dposv3.GetStateRequest{}, &stateResp,
			)
			if err != nil {
				return errors.Wrap(err, "failed to fetch DPOS contract state")
			}

			if stateResp.State.Params.OracleAddress.Local.Compare(signerAddr) != 0 {
				return errors.New("private key doesn't match oracle")
			}

			if len(stateResp.State.Validators) < 2 {
				return errors.New("insufficient number of validators")
			}

			var candidatesResp dposv3.ListCandidatesResponse
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListCandidates",
				&dposv3.ListCandidatesRequest{}, &candidatesResp,
			)
			if err != nil {
				return errors.Wrap(err, "failed to fetch candidates")
			}

			type redelegationInfo struct {
				Delegator   string
				FromIndex   uint64
				ToValidator string
				Amount      string
			}
			type validatorInfo struct {
				Name    string
				Address string
			}
			summary := struct {
				FromValidator string
				ToValidators  []validatorInfo
				Redelegations []redelegationInfo
			}{
				FromValidator: srcValidatorAddr.String(),
			}

			// if the validator is no longer a candidate there's no way to figure out what their
			// fee was so just assume it was 100%
			currentFee := uint64(10000)
			for _, c := range candidatesResp.Candidates {
				if srcValidatorAddr.Compare(loom.UnmarshalAddressPB(c.Candidate.Address)) == 0 {
					currentFee = c.Candidate.Fee
					break
				}
			}

			// make a list of all the other validators we'll be redelegating to
			validators := make([]loom.Address, 0, len(stateResp.State.Validators))
			for _, v := range stateResp.State.Validators {
				validatorAddr := loom.Address{
					ChainID: flags.ChainID,
					Local:   loom.LocalAddressFromPublicKey(v.PubKey),
				}
				if validatorAddr.Compare(srcValidatorAddr) == 0 {
					continue
				}
				// only redelegate to validators with the same or lower fee as the original validator
				for _, c := range candidatesResp.Candidates {
					if validatorAddr.Compare(loom.UnmarshalAddressPB(c.Candidate.Address)) == 0 {
						if !c.Statistic.Jailed && (c.Candidate.Fee <= currentFee) {
							validators = append(validators, validatorAddr)
							summary.ToValidators = append(summary.ToValidators, validatorInfo{
								Name:    c.Candidate.Name,
								Address: validatorAddr.String(),
							})
						}
						break
					}
				}
			}

			var delegationsResp dposv3.ListDelegationsResponse
			err = cli.StaticCallContractWithFlags(
				&flags, DPOSV3ContractName, "ListDelegations",
				&dposv3.ListDelegationsRequest{Candidate: srcValidatorAddr.MarshalPB()}, &delegationsResp,
			)
			if err != nil {
				return errors.Wrap(err, "failed to fetch delegations")
			}

			destValidatorPerDelegator := map[string]loom.Address{}

			// randomly redelegate all the validator's delegations
			rand.Seed(time.Now().UTC().UnixNano())
			for _, d := range delegationsResp.Delegations {
				// don't bother redelegating worthless delegations
				if d.State != dposv3.Delegation_BONDED || d.Amount == nil || d.Amount.Value.Int.Sign() <= 0 {
					continue
				}
				delegatorAddr := loom.UnmarshalAddressPB(d.Delegator)
				// don't redelegate the validator rewards delegation
				if d.Index == 0 && delegatorAddr.Compare(srcValidatorAddr) == 0 {
					continue
				}

				// if a delegator has multiple delegations to the original validator move them all to
				// a single validator instead of dispersing them across all the other validators,
				// this will allow the delegator to easily consolidate them all later if they want
				var destValidatorAddr loom.Address
				var exists bool
				if destValidatorAddr, exists = destValidatorPerDelegator[delegatorAddr.String()]; !exists {
					destValidatorAddr = validators[rand.Intn(len(validators))]
					destValidatorPerDelegator[delegatorAddr.String()] = destValidatorAddr
				}

				summary.Redelegations = append(summary.Redelegations, redelegationInfo{
					Delegator:   delegatorAddr.String(),
					FromIndex:   d.Index,
					ToValidator: destValidatorAddr.String(),
					Amount:      d.Amount.Value.Int.String(),
				})

				err = cli.CallContractWithFlags(
					&flags, DPOSV3ContractName, "Redelegate",
					&dposv3.RedelegateRequest{
						DelegatorAddress:       d.Delegator,
						Index:                  d.Index,
						ValidatorAddress:       destValidatorAddr.MarshalPB(),
						FormerValidatorAddress: srcValidatorAddr.MarshalPB(),
					}, nil,
				)
				if err != nil {
					out, marshalErr := json.MarshalIndent(summary, "", "  ")
					if marshalErr != nil {
						return err
					}
					fmt.Println(string(out))
					return errors.Wrapf(
						err, "failed to redelegate delegation at index %d to %s",
						d.Index, destValidatorAddr.String(),
					)
				}
			}
			out, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}
	cli.AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

const claimDelegatorRewardsCmdExample = `
loom dpos3 claim-delegator-rewards --key path/to/private_key
`

func ClaimDelegatorRewardsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "claim-delegator-rewards",
		Short:   "claim pending delegation rewards",
		Example: claimDelegatorRewardsCmdExample,
		Args:    cobra.MinimumNArgs(0),
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

const checkDelegatorRewardsCmdExample = `
loom dpos3 check-delegator-rewards 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func CheckDelegatorRewardsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "check-delegator-rewards <address>",
		Short:   "check rewards for the specified delegator",
		Example: checkDelegatorRewardsCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const checkRewardCmdExample = `
loom dpos3 check-rewards -u http://localhost:12345
`

func CheckRewardsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "check-rewards",
		Short:   "check rewards statistics",
		Example: checkRewardCmdExample,
		Args:    cobra.MinimumNArgs(0),
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

const checkAllDelegationsCmdExample = `
loom dpos3 check-all-delegations 0x7262d4c97c7B93937E4810D289b7320e9dA82857
`

func CheckAllDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "check-all-delegations [delegator]",
		Short:   "display all of a particular delegator's delegations",
		Example: checkAllDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const timeUntilElectionCmdExample = `
loom dpos3 time-until-election -u http://localhost:12345
`

func TimeUntilElectionCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "time-until-election",
		Short:   "check how many seconds remain until the next election",
		Example: timeUntilElectionCmdExample,
		Args:    cobra.MinimumNArgs(0),
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

const listDelegationsCmdExample = `
loom dpos3 list-delegations 0x7262d4c97c7B93937E4810D289b7320e9dA82857 --concise
`

func ListDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	var conciseMode bool
	cmd := &cobra.Command{
		Use:     "list-delegations <candidate address>",
		Short:   "list a candidate's delegations & delegation total",
		Example: listDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

			if conciseMode {
				type delegationInfo struct {
					Delegator    string              `json:"delegator"`
					Validator    string              `json:"validator"`
					Index        uint64              `json:"index"`
					Amount       string              `json:"amount"`
					LocktimeTier dposv3.LocktimeTier `json:"locktimeTier"`
					LockTime     uint64              `json:"lockTime"`
					State        string              `json:"state"`
				}
				type outputInfo struct {
					Delegations     []*delegationInfo `json:"delegations"`
					DelegationTotal string            `json:"delegationTotal"`
				}
				delegations := []*delegationInfo{}
				for _, d := range resp.Delegations {
					delegations = append(delegations, &delegationInfo{
						Delegator:    loom.UnmarshalAddressPB(d.Delegator).Local.String(),
						Validator:    loom.UnmarshalAddressPB(d.Validator).Local.String(),
						Index:        d.Index,
						Amount:       d.Amount.Value.Int.String(),
						LocktimeTier: d.LocktimeTier,
						LockTime:     d.LockTime,
						State:        d.State.String(),
					})
				}
				output := outputInfo{
					Delegations:     delegations,
					DelegationTotal: resp.DelegationTotal.Value.Int.String(),
				}
				prettyJSON, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return errors.Wrap(err, "failed to generate json output")
				}
				fmt.Println(string(prettyJSON))
			} else {
				out, err := formatJSON(&resp)
				if err != nil {
					return err
				}
				fmt.Println(out)
			}
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	cmd.Flags().BoolVar(&conciseMode, "concise", false, "Omit less relevant details")
	return cmd
}

const listAllDelegationsCmdExample = `
loom dpos3 list-all-delegations -u http://localhost:12345
`

func ListAllDelegationsCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "list-all-delegations",
		Short:   "display the results of calling list_delegations for all candidates",
		Example: listAllDelegationsCmdExample,
		Args:    cobra.MinimumNArgs(0),
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

const registerReferrerCmdExample = `
loom dpos3 register-referrer referrer_name 0x7262d4c97c7B93937E4810D289b7320e9dA82857 --key path/to/private_key
`

func RegisterReferrerCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "register-referrer [name] [address]",
		Short:   "Register a referrer wallet's name and address",
		Example: registerReferrerCmdExample,
		Args:    cobra.MinimumNArgs(2),
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

const setElectionCycleCmdExample = `
loom dpos3 set-election-cycle 30000 --key path/to/private_key
`

func SetElectionCycleCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-election-cycle [election duration]",
		Short:   "Set election cycle duration (in seconds)",
		Example: setElectionCycleCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const setDowntimePeriodCmdExample = `
loom dpos3 set-downtime-period 4096 --key path/to/private_key
`

func SetDowntimePeriodCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-downtime-period [downtime period]",
		Short:   "Set downtime period duration (in blocks)",
		Example: setDowntimePeriodCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const enableValidatorJailingCmdExample = `
loom dpos3 enable-validator-jailing true -k path/to/private_key
`

func EnableValidatorJailingCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "enable-validator-jailing [enable] ",
		Short:   "Toggle jailing of offline validators",
		Example: enableValidatorJailingCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("Invalid boolean status")
			}
			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "EnableValidatorJailing", &dposv3.EnableValidatorJailingRequest{
					JailOfflineValidators: status,
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

const setValidatorCountCmdExample = `
loom dpos3 set-validator-count 21 --key path/to/private_key
`

func SetValidatorCountCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-validator-count [validator count]",
		Short:   "Set maximum number of validators",
		Example: setValidatorCountCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const setMaxYearlyRewardCmdExample = `
loom dpos3 set-max-yearly-reward 10000 --key path/to/private_key
`

func SetMaxYearlyRewardCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-max-yearly-reward [max yearly rewward amount]",
		Short:   "Set maximum yearly reward",
		Example: setMaxYearlyRewardCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const setRegistrationRequirementCmdExample = `
loom dpos3 set-registration-requirement 100 --key path/to/private_key
`

func SetRegistrationRequirementCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-registration-requirement [registration_requirement]",
		Short:   "Set minimum self-delegation required of a new Candidate",
		Example: setRegistrationRequirementCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const setOracleAddressCmdExample = `
loom dpos3 set-oracle-address 0x7262d4c97c7B93937E4810D289b7320e9dA82857 --key path/to/private_key
`

func SetOracleAddressCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-oracle-address [oracle address]",
		Short:   "Set oracle address",
		Example: setOracleAddressCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const setSlashingPercentagesCmdExample = `
loom dpos3 set-slashing-percentages 100 300 --key path/to/private_key
`

func SetSlashingPercentagesCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-slashing-percentages [crash fault slashing percentage] [byzantine fault slashing percentage",
		Short:   "Set crash and byzantine fualt slashing percentages expressed in basis points",
		Example: setSlashingPercentagesCmdExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			crashFaultSlashingPercentage, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}
			byzantineFaultSlashingPercentage, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetSlashingPercentages", &dposv3.SetSlashingPercentagesRequest{
					CrashSlashingPercentage: &types.BigUInt{
						Value: *loom.NewBigUIntFromInt(crashFaultSlashingPercentage),
					},
					ByzantineSlashingPercentage: &types.BigUInt{
						Value: *loom.NewBigUIntFromInt(byzantineFaultSlashingPercentage),
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

func SetMaxDowntimePercentageCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "set-max-downtime-percentage [max downtime percentage]",
		Short: "Set crash fault downtime percentage expressed in basis points",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxDowntimePercentage, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "SetMaxDowntimePercentage", &dposv3.SetMaxDowntimePercentageRequest{
					MaxDowntimePercentage: &types.BigUInt{
						Value: *loom.NewBigUIntFromInt(maxDowntimePercentage),
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

const setMinCandidateFeeCmdExample = `
loom dpos3 set-min-candidate-fee 900 --key path/to/private_key
`

func SetMinCandidateFeeCmdV3() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "set-min-candidate-fee [min candidate fee]",
		Short:   "Set minimum candidate fee",
		Example: setMinCandidateFeeCmdExample,
		Args:    cobra.MinimumNArgs(1),
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

const ignoreUnbondLocktimeCmdExample = `
loom dpos3 ignore-unbond-locktime true -k path/to/private_key
`

func IgnoreUnbondLocktimeCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:     "ignore-unbond-locktime [true|false]",
		Example: ignoreUnbondLocktimeCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("invalid boolean value")
			}
			err = cli.CallContractWithFlags(
				&flags, DPOSV3ContractName, "IgnoreUnbondLocktime", &dposv3.IgnoreUnbondLocktimeRequest{
					Ignore: status,
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
		ListReferrersCmdV3(),
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
		UnbondAllDelegationsCmdV3(),
		RedelegateAllDelegationsCmd(),
		RegisterReferrerCmdV3(),
		SetDowntimePeriodCmdV3(),
		SetElectionCycleCmdV3(),
		SetValidatorCountCmdV3(),
		SetMaxYearlyRewardCmdV3(),
		SetRegistrationRequirementCmdV3(),
		SetOracleAddressCmdV3(),
		SetSlashingPercentagesCmdV3(),
		SetMaxDowntimePercentageCmdV3(),
		ChangeFeeCmdV3(),
		TimeUntilElectionCmdV3(),
		GetStateCmdV3(),
		SetMinCandidateFeeCmdV3(),
		UnjailValidatorCmdV3(),
		EnableValidatorJailingCmd(),
		IgnoreUnbondLocktimeCmd(),
	)
	return cmd
}
