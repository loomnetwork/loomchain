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
	defaultStorageMethod = ctypes.ReceiptsStorage_LEVELDB
	defaultMaxReceiptStorage = 0
)

var (
	Keys = map[ctypes.ConfigParamter][]byte{
		ctypes.ConfigParamter_ORACLE:          []byte("oracle"),
		ctypes.ConfigParamter_RECEIPT_STORAGE: []byte("config:Receipt:Storage"),
		ctypes.ConfigParamter_RECEIPT_MAX:     []byte("config:Receipts:max"),
	}
	oracleRole = []string{"oracle"}
	oldOracleRole = []string{"old-oracle"}
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
	var method ctypes.ReceiptsStorageMethod
	var max ctypes.ReceiptsMax
	if req.Receipts == nil {
		method.StorageMethod = defaultStorageMethod
		max.Max = defaultMaxReceiptStorage
	} else {
		method = ctypes.ReceiptsStorageMethod{req.Receipts.StorageMethod}
		max = ctypes.ReceiptsMax{req.Receipts.Max}
	}
	if err := ctx.Set(
		Keys[ctypes.ConfigParamter_RECEIPT_STORAGE],
		&ctypes.ConfigValue{&ctypes.ConfigValue_ReceiptsStorageMethod{&method}},
	); err != nil {
		return errors.Wrap(err, "set receipt storage method")
	}
	if err := ctx.Set(
		Keys[ctypes.ConfigParamter_RECEIPT_MAX],
		&ctypes.ConfigValue{&ctypes.ConfigValue_ReceiptsMax{&max}},
	); err != nil {
		return errors.Wrap(err, "set max receipts stored")
	}

	if req.Oracle != nil {
		ctx.GrantPermissionTo(loom.UnmarshalAddressPB(req.Oracle), []byte(req.Oracle.String()), "oracle")
		if err := ctx.Set(
			Keys[ctypes.ConfigParamter_ORACLE],
			&ctypes.ConfigValue{&ctypes.ConfigValue_NewOracle{req.Oracle}},
		); err != nil {
			return errors.Wrap(err, "setting oracle")
		}
	}
	
	return nil
}

func (c *Config) Set(ctx contractpb.Context, param *ctypes.SetParam) error {
	valueType :=  getParamType(*param.Value)
	if valueType == ctypes.ConfigParamter_UNKNOWN {
		return errors.New("unknown value type")
	}
	if valueType == ctypes.ConfigParamter_ORACLE {
		return c.setOracle(ctx, param)
	}
	if err := c.validateOracle(ctx, param.Oracle); err != nil {
		return errors.Wrap(err, "validating oracle")
	}
	if err := ctx.Set(Keys[valueType], param.Value); err != nil {
		return errors.Wrapf(err, "saving value to state", )
	}
	return nil
}

func (c *Config) Get(ctx contractpb.StaticContext, valueType ctypes.ValueType ) (*ctypes.ConfigValue, error) {
	if valueType.Type == ctypes.ConfigParamter_UNKNOWN {
		return nil, errors.New("invalid parmeter")
	}
	var value ctypes.ConfigValue
	if err := ctx.Get(Keys[valueType.Type], &value); err != nil {
		// Some stores (eg some mock ones) treat setting to zero value as deleting.
		// All keys should be set to something in Init().
		// So treat "not found" as the zero value
		if err.Error() == "not found" {
			var zeroValue ctypes.ConfigValue
			return &zeroValue, nil
		} else {
			return nil, errors.Wrap(err,"get value ")
		}
	}
	return &value, nil
}

func getParamType(value ctypes.ConfigValue) ctypes.ConfigParamter {
	switch value.Value.(type) {
	case *ctypes.ConfigValue_NewOracle: return ctypes.ConfigParamter_ORACLE
	case *ctypes.ConfigValue_ReceiptsStorageMethod: return ctypes.ConfigParamter_RECEIPT_STORAGE
	case *ctypes.ConfigValue_ReceiptsMax: return ctypes.ConfigParamter_RECEIPT_MAX
	default: return ctypes.ConfigParamter_UNKNOWN
	}
}

func (c *Config) setOracle(ctx contractpb.Context, params *ctypes.SetParam) error {
	newOracle := params.Value.GetNewOracle()
	if len(newOracle.Local) <= 0 {
		return errors.New("missing new oracle")
	}
	if ctx.Has(Keys[ctypes.ConfigParamter_ORACLE]) {
		if err := c.validateOracle(ctx, params.Oracle); err != nil {
			return errors.Wrap(err, "validating oracle")
		}
		ctx.GrantPermission([]byte(params.Oracle.String()), oldOracleRole)
	}
	ctx.GrantPermission([]byte(newOracle.String()), oracleRole)
	
	if err := ctx.Set(Keys[ctypes.ConfigParamter_ORACLE], newOracle); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

func (c *Config) validateOracle(ctx contractpb.Context, ko *types.Address) error {
	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"oracle"}); !ok {
		if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"old-oracle"}); ok {
			return errors.New("oracle has expired")
		} else {
			return errors.New("oracle unverified")
		}
	}
	return nil
}

var Contract plugin.Contract = contractpb.MakePluginContract(&Config{})