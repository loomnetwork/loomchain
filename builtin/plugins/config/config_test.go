package config

import (
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/plugin`
	`github.com/loomnetwork/go-loom/plugin/contractpb`
	`github.com/stretchr/testify/require`
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	`testing`
)
var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()
	oracle  = types_addr1
	user    = types_addr2
)


func TestConfigInit(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Config{}
	err := contract.Init(ctx, &ctypes.ConfigInitRequest{
		Oracle:  oracle,
		Receipts: &ctypes.Receipts{
			StorageMethod: ctypes.ReceiptsStorage_LEVELDB,
			Max: 98,
		},
	})
	require.NoError(t, err)
	
	method, err := contract.Get(ctx, ctypes.ValueType{ctypes.ConfigParamter_RECEIPT_STORAGE})
	require.NoError(t, err)
	require.Equal(t, method.GetReceiptsStorageMethod().StorageMethod, ctypes.ReceiptsStorage_LEVELDB)
	
	max, err := contract.Get(ctx,ctypes.ValueType{ctypes.ConfigParamter_RECEIPT_MAX})
	require.NoError(t, err)
	require.Equal(t, max.GetReceiptsMax().Max, uint64(98))
}

func TestMethods(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Config{}
	err := contract.Init(ctx, &ctypes.ConfigInitRequest{
		Oracle: oracle,
		Receipts: &ctypes.Receipts{
			StorageMethod: ctypes.ReceiptsStorage_LEVELDB,
			Max: 98,
		},
	})
	require.NoError(t, err)
	
	method, err := contract.Get(ctx, ctypes.ValueType{ctypes.ConfigParamter_RECEIPT_STORAGE})
	require.NoError(t, err)
	require.Equal(t, ctypes.ReceiptsStorage_LEVELDB, method.GetReceiptsStorageMethod().StorageMethod)
	
	methodValue := ctypes.ConfigValue_ReceiptsStorageMethod{
		&ctypes.ReceiptsStorageMethod{ctypes.ReceiptsStorage_CHAIN},
	}
	require.NoError(t, contract.Set(ctx, &ctypes.SetParam{
		oracle,
		&ctypes.ConfigValue{&methodValue},
	}))
	
	method, err = contract.Get(ctx, ctypes.ValueType{ctypes.ConfigParamter_RECEIPT_STORAGE})
	require.NoError(t, err)
	require.Equal(t, ctypes.ReceiptsStorage_CHAIN, method.GetReceiptsStorageMethod().StorageMethod)
	
	max, err := contract.Get(ctx, ctypes.ValueType{ctypes.ConfigParamter_RECEIPT_MAX})
	require.NoError(t, err)
	require.Equal(t,uint64(98), max.GetReceiptsMax().Max )
	
	maxValue := ctypes.ConfigValue_ReceiptsMax{&ctypes.ReceiptsMax{uint64(50)}}
	require.NoError(t, contract.Set(ctx, &ctypes.SetParam{
		oracle,
		&ctypes.ConfigValue{&maxValue},
	}))
	
	max, err = contract.Get(ctx, ctypes.ValueType{ctypes.ConfigParamter_RECEIPT_MAX})
	require.NoError(t, err)
	require.Equal(t, uint64(50), max.GetReceiptsMax().Max)
	
}

func TestKarmaValidateOracle(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Config{}
	err := contract.Init(ctx, &ctypes.ConfigInitRequest{
		Oracle: oracle,
	})
	require.NoError(t, err)
	
	err = contract.validateOracle(ctx, oracle)
	require.NoError(t, err)
	
	err = contract.validateOracle(ctx, user)
	require.Error(t, err)
	
}