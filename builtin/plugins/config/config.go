package config

import (
	`github.com/loomnetwork/go-loom`
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	`github.com/loomnetwork/go-loom/plugin`
	`github.com/loomnetwork/go-loom/plugin/contractpb`
	"github.com/loomnetwork/go-loom/types"
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
	
	valueTypes = map[string]string{
		ConfigKeyOracle:        "Address",
		ConfigKeyRecieptStrage: "ReceiptStorage",
		ConfigKeyReceiptMax:    "uint64",
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
	for _, kv := range req.Settings {
		ctx.Set([]byte("config:" + kv.Key), kv.Value)
	}

	if req.Oracle != nil {
		ctx.GrantPermissionTo(loom.UnmarshalAddressPB(req.Oracle), []byte(req.Oracle.String()), "oracle")
		if err := ctx.Set([]byte("oracle"),	req.Oracle); err != nil {
			return errors.Wrap(err, "setting oracle")
		}
	}
	
	return nil
}

func (c *Config) Set(ctx contractpb.Context, param *ctypes.SetKeyValue) error {
	if param.Key == "oracle" {
		return setOracle(ctx, param)
	}
	if err := validateOracle(ctx, param.Oracle); err != nil {
		return err
	}
	if err := validateValue(param.Key, param.Value.Data); err != nil {
		return err
	}
	if err := ctx.Set(stateKey(param.Key), param.Value); err != nil {
		return errors.Wrapf(err, "saving value to state", )
	}
	return nil
}

func (c *Config) Get(ctx contractpb.StaticContext, key ctypes.Key ) (*ctypes.Value, error) {
	var value ctypes.Value
	if err := ctx.Get(stateKey(key.Key), &value); err != nil {
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

func setOracle(ctx contractpb.Context, params *ctypes.SetKeyValue) error {
	newOracle := params.Value.GetAddress()
	if len(newOracle.Local) <= 0 {
		return errors.New("missing new oracle")
	}
	if ctx.Has([]byte(ConfigKeyOracle)) {
		if err := validateOracle(ctx, params.Oracle); err != nil {
			return errors.Wrap(err, "validating oracle")
		}
		ctx.GrantPermission([]byte(params.Oracle.String()), oldOracleRole)
	}
	ctx.GrantPermission([]byte(newOracle.String()), oracleRole)
	
	if err := ctx.Set([]byte(ConfigKeyOracle), newOracle); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

func validateValue(key string, value interface{}) error {
	if _, ok := valueTypes[key]; !ok {
		return errors.New("unrecognised key")
	}
	switch  value.(type) {
	case *ctypes.Value_Uint64Val:
		if valueTypes[key] != "uint64" {
			return errors.New("mismatched type")
		}
	case *ctypes.Value_ReceiptStorage:
		if valueTypes[key] != "ReceiptStorage" {
			return errors.New("mismatched type")
		}
	case *ctypes.Value_Address:
		if valueTypes[key] != "Address" {
			return errors.New("mismatched type")
		}
	default:
		return errors.Errorf("mismatched type")
	}
	return nil
}

func validateOracle(ctx contractpb.Context, ko *types.Address) error {
	if ok, _ := ctx.HasPermission([]byte(ko.String()), oracleRole); !ok {
		if ok, _ := ctx.HasPermission([]byte(ko.String()), oldOracleRole); ok {
			return errors.New("oracle has expired")
		} else {
			return errors.New("oracle unverified")
		}
	}
	return nil
}

func stateKey(k string) []byte {
	return []byte("config:" + k)
}

var Contract plugin.Contract = contractpb.MakePluginContract(&Config{})