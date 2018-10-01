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
		Settings: []*ctypes.KeyValue{
			{ConfigKeyRecieptStrage, &ctypes.Value{
				&ctypes.Value_ReceiptStorage{ctypes.ReceiptStorage_LEVELDB },
			}},
			{ConfigKeyReceiptMax, &ctypes.Value{
				&ctypes.Value_Uint64Val{ uint64(98)},
			}},
		},
	})
	require.NoError(t, err)
	
	method, err := contract.Get(ctx, ctypes.Key{ConfigKeyRecieptStrage})
	require.NoError(t, err)
	require.Equal(t, method.GetReceiptStorage(), ctypes.ReceiptStorage_LEVELDB)
	
	max, err := contract.Get(ctx,ctypes.Key{ConfigKeyReceiptMax})
	require.NoError(t, err)
	require.Equal(t, max.GetUint64Val(), uint64(98))
}

func TestMethods(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	contract := &Config{}
	err := contract.Init(ctx, &ctypes.ConfigInitRequest{
		Oracle:  oracle,
		Settings: []*ctypes.KeyValue{
			{ConfigKeyRecieptStrage, &ctypes.Value{
				&ctypes.Value_ReceiptStorage{ctypes.ReceiptStorage_LEVELDB },
			}},
			{ConfigKeyReceiptMax, &ctypes.Value{
				&ctypes.Value_Uint64Val{ uint64(98)},
			}},
		},
	})
	require.NoError(t, err)
	
	method, err := contract.Get(ctx, ctypes.Key{"receipt-storage"})
	require.NoError(t, err)
	require.Equal(t, method.GetReceiptStorage(), ctypes.ReceiptStorage_LEVELDB)
	
	methodValue := ctypes.Value_ReceiptStorage{
		ctypes.ReceiptStorage_CHAIN,
	}
	require.NoError(t, contract.Set(ctx, &ctypes.SetKeyValue{
		oracle,
		ConfigKeyRecieptStrage,
		&ctypes.Value{&methodValue},
	}))
	
	method, err = contract.Get(ctx, ctypes.Key{ConfigKeyRecieptStrage})
	require.NoError(t, err)
	require.Equal(t, method.GetReceiptStorage(), ctypes.ReceiptStorage_CHAIN)
	
	max, err := contract.Get(ctx,ctypes.Key{ConfigKeyReceiptMax})
	require.NoError(t, err)
	require.Equal(t,uint64(98), max.GetUint64Val() )
	
	maxValue := ctypes.Value_Uint64Val{uint64(50)}
	require.NoError(t, contract.Set(ctx, &ctypes.SetKeyValue{
		oracle,
		"receipt-max",
		&ctypes.Value{&maxValue},
	}))
	
	max, err = contract.Get(ctx,ctypes.Key{ConfigKeyReceiptMax})
	require.NoError(t, err)
	require.Equal(t, uint64(50), max.GetUint64Val())
	
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
	
	err = validateOracle(ctx, oracle)
	require.NoError(t, err)
	
	err = validateOracle(ctx, user)
	require.Error(t, err)
	
}