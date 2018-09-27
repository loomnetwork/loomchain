package config

import (
	`github.com/loomnetwork/go-loom`
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	`github.com/loomnetwork/go-loom/plugin`
	`github.com/loomnetwork/go-loom/plugin/contractpb`
	"github.com/loomnetwork/go-loom/types"
	`github.com/pkg/errors`
)

var (
	oracleKey       = []byte("oracle")
	
	receiptStorageKey  = []byte("config:ReceiptStorage")
	MaxReceiptsKey     = []byte("config:MaxReceipts")
	
	oracleRole = []string{"oracle"}
	oldOracleRole = []string{"old-oracle"}
)

type Config struct {
}

func (k *Config) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "config",
		Version: "1.0.0",
	}, nil
}

func (k *Config) Init(ctx contractpb.Context, req *ctypes.ConfigInitRequest) error {
	if req.Receipts != nil {
		method := ctypes.ReceiptStorageMethod{req.Receipts.StorageMethod}
		if err := ctx.Set(receiptStorageKey, &method); err != nil {
			return errors.Wrap(err, "set receipt storage method")
		}
		
		max := ctypes.MaxReceipts{req.Receipts.MaxReceipts}
		if err := ctx.Set(MaxReceiptsKey, &max); err != nil {
			return errors.Wrap(err, "set max receipts stored")
		}
	}

	if req.Oracle != nil {
		ctx.GrantPermissionTo(loom.UnmarshalAddressPB(req.Oracle), []byte(req.Oracle.String()), "oracle")
		if err := ctx.Set(oracleKey, req.Oracle); err != nil {
			return errors.Wrap(err, "error setting oracle")
		}
	}
	
	return nil
}

func (k *Config) SetOracle(ctx contractpb.Context, params *ctypes.NewOracle) error {
	if ctx.Has(oracleKey) {
		if err := k.validateOracle(ctx, params.OldOracle); err != nil {
			return errors.Wrap(err, "validating oracle")
		}
		ctx.GrantPermission([]byte(params.OldOracle.String()), oldOracleRole)
	}
	ctx.GrantPermission([]byte(params.NewOracle.String()), oracleRole)
	
	if err := ctx.Set(oracleKey, params.NewOracle); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

func (k *Config) SetReceiptStorageMethod(ctx contractpb.Context, setMethod *ctypes.NewReceiptStorageMethod) error {
	if err := k.validateOracle(ctx, setMethod.Oracle); err != nil {
		return errors.Wrap(err, "validating oracle")
	}
	if err := ctx.Set(receiptStorageKey,&ctypes.ReceiptStorageMethod{setMethod.NewStorageMethod}); err != nil {
		return errors.Wrap(err, "Error setting storage method")
	}
	return nil
}

func (k *Config) SetMaxReceipts(ctx contractpb.Context, setMax *ctypes.NewMaxReceipts) error {
	if err := k.validateOracle(ctx, setMax.Oracle); err != nil {
		return errors.Wrap(err, "validating oracle")
	}
	if err := ctx.Set(receiptStorageKey,&ctypes.MaxReceipts{setMax.MaxReceipts}); err != nil {
		return errors.Wrap(err, "Error setting storage method")
	}
	return nil
}

func (k *Config) GetReceiptStorageMethod(ctx contractpb.StaticContext) (*ctypes.ReceiptStorageMethod, error) {
	var method ctypes.ReceiptStorageMethod
	if err := ctx.Get(receiptStorageKey, &method); err != nil {
		return nil, err
	}
	return &method, nil
}

func (k *Config) GetMaxReceipts(ctx contractpb.StaticContext) (*ctypes.MaxReceipts, error) {
	var max ctypes.MaxReceipts
	if err := ctx.Get(MaxReceiptsKey, &max); err != nil {
		return nil, err
	}
	return &max, nil
}

func (k *Config) validateOracle(ctx contractpb.Context, ko *types.Address) error {
	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"oracle"}); !ok {
		return errors.New("Oracle unverified")
	}
	
	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"old-oracle"}); ok {
		return errors.New("This oracle is expired. Please use latest oracle.")
	}
	return nil
}

var Contract plugin.Contract = contractpb.MakePluginContract(&Config{})