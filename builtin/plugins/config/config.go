package config

import (
	`github.com/loomnetwork/go-loom`
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	`github.com/loomnetwork/go-loom/plugin`
	`github.com/loomnetwork/go-loom/plugin/contractpb`
	`github.com/pkg/errors`
)

const (
	ConfigKeyOracle = "oracle"
	ConfigKeyRecieptStrage = "receipt-storage"
	ConfigKeyReceiptMax = "receipt-max"
)

var (
	oracleRole =    []string{"oracle"}
	oldOracleRole = []string{"old-oracle"}
	
	ValueTypes = map[string]string{
		ConfigKeyOracle:        "Value_Address",
		ConfigKeyRecieptStrage: "Value_ReceiptStorage",
		ConfigKeyReceiptMax:    "Value_Uint64Val",
	}
)

type Config struct {
}

func (c *Config) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "config",
		Version: "1.0.0",
	}, nil
}

func (c *Config) Init(ctx contractpb.Context, req *ctypes.ConfigInitRequest) error {
	if req.Oracle != nil {
		oracle := loom.UnmarshalAddressPB(req.Oracle)
		ctx.GrantPermissionTo(oracle, []byte(oracle.String()), "oracle")
		
		oracleValue := ctypes.Value{&ctypes.Value_Address{req.Oracle}}
		if err := ctx.Set(StateKey(ConfigKeyOracle), &oracleValue); err != nil {
			return errors.Wrap(err, "setting oracle")
		}
	}
	
	for _, kv := range req.Settings {
		if (kv.Key != ConfigKeyOracle) {
			err := validateValue(kv.Key, kv.Value.Data)
			if err != nil {
				return errors.Wrapf(err, "validating config key %s", kv.Key)
			}
			err = ctx.Set(StateKey(kv.Key), kv.Value)
			if err != nil {
				return errors.Wrapf(err, "saving config key %s", kv.Key)
			}
		} else {
			return errors.New("set oracle separately")
		}
	}
	
	return nil
}

func (c *Config) Set(ctx contractpb.Context, param *ctypes.UpdateSetting) error {
	if param.Key == ConfigKeyOracle {
		return setOracle(ctx, param)
	}
	if err := validateOracle(ctx); err != nil {
		return err
	}
	if err := validateValue(param.Key, param.Value.Data); err != nil {
		return err
	}
	if err := ctx.Set(StateKey(param.Key), param.Value); err != nil {
		return errors.Wrapf(err, "saving value to state", )
	}
	return nil
}

func (c *Config) Get(ctx contractpb.StaticContext, key *ctypes.GetSetting ) (*ctypes.Value, error) {
	var value ctypes.Value
	if err := ctx.Get(StateKey(key.Key), &value); err != nil {
		// Some stores (eg some mock ones) treat setting to zero value as deleting.
		// So treat "not found" as the zero value
		if err.Error() == "not found" {
			var zeroValue ctypes.Value
			return &zeroValue, nil
		} else {
			return nil, errors.Wrap(err,"get value ")
		}
	}
	return &value, nil
}

func setOracle(ctx contractpb.Context, params *ctypes.UpdateSetting) error {
	newOracle := loom.UnmarshalAddressPB(params.Value.GetAddress())
	if len(newOracle.Local) <= 0 {
		return errors.New("missing new oracle")
	}
	if ctx.Has([]byte(ConfigKeyOracle)) {
		if err := validateOracle(ctx); err != nil {
			return errors.Wrap(err, "validating oracle")
		}
		ctx.GrantPermission([]byte(ctx.Message().Sender.String()), oldOracleRole)
		ctx.RevokePermissionFrom(ctx.Message().Sender, []byte(ctx.Message().Sender.String()), oracleRole[0])
	}
	ctx.GrantPermission([]byte(newOracle.String()), oracleRole)
	if err := ctx.Set([]byte(ConfigKeyOracle), params.Value); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

func validateValue(key string, value interface{}) error {
	if _, ok := ValueTypes[key]; !ok {
		return errors.Errorf("unrecognised key, %s", key)
	}
	switch  value.(type) {
	case *ctypes.Value_Uint64Val:
		if ValueTypes[key] != "Value_Uint64Val" {
			return errors.Errorf("mismatched type, exected %s", ValueTypes[key])
		}
	case *ctypes.Value_ReceiptStorage:
		if ValueTypes[key] != "Value_ReceiptStorage" {
			return errors.Errorf("mismatched type, exected %s", ValueTypes[key])
		}
	case *ctypes.Value_Address:
		if ValueTypes[key] != "Value_Address" {
			return errors.Errorf("mismatched type, exected %s", ValueTypes[key])
		}
	default:
		return errors.Errorf("mismatched type, exected %s", ValueTypes[key])
	}
	return nil
}

func validateOracle(ctx contractpb.Context) error {
	caller := ctx.Message().Sender
	if ok, _ := ctx.HasPermission([]byte(caller.String()), oracleRole); !ok {
		if ok, _ := ctx.HasPermission([]byte(caller.String()), oldOracleRole); ok {
			return errors.New("oracle has expired")
		} else {
			return errors.New("oracle unverified")
		}
	}
	return nil
}

func StateKey(k string) []byte {
	if k != ConfigKeyOracle {
		return []byte("config:" + k)
	} else {
		return []byte(ConfigKeyOracle)
	}
	
}

var Contract plugin.Contract = contractpb.MakePluginContract(&Config{})