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
			StorageMethod: ctypes.ReceiptStorage_LEVELDB,
			MaxReceipts: 98,
		},
	})
	require.NoError(t, err)
	
	method, err := contract.GetReceiptStorageMethod(ctx)
	require.NoError(t, err)
	require.Equal(t, method.StorageMethod, ctypes.ReceiptStorage_LEVELDB)
	
	max, err := contract.GetMaxReceipts(ctx)
	require.NoError(t, err)
	require.Equal(t, max.MaxReceipts, uint64(98))
}

func TestMethods(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Config{}
	err := contract.Init(ctx, &ctypes.ConfigInitRequest{
		Oracle: oracle,
		Receipts: &ctypes.Receipts{
			StorageMethod: ctypes.ReceiptStorage_LEVELDB,
			MaxReceipts: 98,
		},
	})
	require.NoError(t, err)
	
	method, err := contract.GetReceiptStorageMethod(ctx)
	require.NoError(t, err)
	require.Equal(t, method.StorageMethod, ctypes.ReceiptStorage_LEVELDB)
	require.NoError(t, contract.SetReceiptStorageMethod(ctx, &ctypes.NewReceiptStorageMethod{ ctypes.ReceiptStorage_CHAIN,oracle}))
	method, err = contract.GetReceiptStorageMethod(ctx)
	require.NoError(t, err)
	require.Equal(t, method.StorageMethod, ctypes.ReceiptStorage_CHAIN)
	
	max, err := contract.GetMaxReceipts(ctx)
	require.NoError(t, err)
	require.Equal(t, max.MaxReceipts, uint64(98))
	require.NoError(t, contract.SetMaxReceipts(ctx, &ctypes.NewMaxReceipts{ uint64(50),oracle}))
	max, err = contract.GetMaxReceipts(ctx)
	require.NoError(t, err)
	require.Equal(t, max.MaxReceipts, uint64(50))
	
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