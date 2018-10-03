package main

import (
	`fmt`
	`github.com/loomnetwork/loomchain/builtin/plugins/config`
	`github.com/pkg/errors`
	`github.com/spf13/cobra`
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	`strconv`
)

const (
	ConfigContractName = "config"
)

func GetSettingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get (setting)",
		Short: "get config setting",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := ctypes.GetSetting{	args[0]	}
			var resp ctypes.Value
			err := staticCallContract(ConfigContractName, "Get", &key, &resp)
			
			if err != nil {
				return errors.Wrap(err, "static call Get")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err,"format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}

func SetSettingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set (setting) (input)",
		Short: "set config setting",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := getConfigValueParmete(args)
			if err != nil {
				return err
			}
			update := ctypes.UpdateSetting{args[0],&value,}
			if err := callContract(ConfigContractName, "Set", &update, nil); err != nil {
				return errors.Wrap(err, "call contract")
			}
			fmt.Println("config setting successfully updated")
			return nil
		},
	}
}

func getConfigValueParmete(args []string) (ctypes.Value, error) {
	if _, ok := config.ValueTypes[args[0]]; !ok {
		return ctypes.Value{}, errors.Errorf("unrecognised setting, %s", args[0])
	}
	var value ctypes.Value
	
	switch config.ValueTypes[args[0]] {
	case "Value_Uint64Val":
		data, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return ctypes.Value{}, errors.Errorf("cannot convert %s to uint64", args[1])
		}
		value.Data = &ctypes.Value_Uint64Val{data}
	case "Value_ReceiptStorage":
		data, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return ctypes.Value{}, errors.Errorf("cannot convert %s to uint64", args[1])
		}
		value.Data = &ctypes.Value_ReceiptStorage{ctypes.ReceiptStorage(data)}
	case "Value_Address":
		user, err := resolveAddress(args[0])
		if err != nil {
			return ctypes.Value{}, errors.Wrap(err, "resolve address arg")
		}
		value.Data = &ctypes.Value_Address{user.MarshalPB()}
	default: return ctypes.Value{}, errors.Errorf("unknown type, %s", config.ValueTypes[args[0]])
	}
	return value, nil
}

func AddConfigMethods(configCmd *cobra.Command) {
	configCmd.AddCommand(
		GetSettingCmd(),
		SetSettingCmd(),
	)
}