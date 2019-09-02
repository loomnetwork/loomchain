package main

import (
	"bytes"
	"encoding/json"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	cconfig "github.com/loomnetwork/go-loom/config"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/plugin"
)

func marshalInit(pb proto.Message) (json.RawMessage, error) {
	var buf bytes.Buffer
	marshaler, err := contractpb.MarshalerFactory(plugin.EncodingType_JSON)
	if err != nil {
		return nil, err
	}
	err = marshaler.Marshal(&buf, pb)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(buf.Bytes()), nil
}

func defaultGenesis(cfg *config.Config, validator *loom.Validator) (*config.Genesis, error) {
	chainID := cfg.ChainID
	if len(chainID) == 0 {
		chainID = "default"
	}

	contracts := []config.ContractConfig{
		{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "coin",
			Location:   "coin:1.0.0",
		},
	}

	if cfg.DPOSVersion == 2 {
		dposV2Init, err := marshalInit(&dposv2.InitRequest{
			Params: &dposv2.Params{
				ValidatorCount:      21,
				ElectionCycleLength: 604800, // one week
			},
			Validators: []*loom.Validator{
				validator,
			},
		})
		if err != nil {
			return nil, err
		}

		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "dposV2",
			Location:   "dposV2:2.0.0",
			Init:       dposV2Init,
		})
	} else if cfg.DPOSVersion == 3 {
		oracleAddr := &types.Address{
			ChainId: chainID,
			Local:   loom.LocalAddressFromPublicKey(validator.PubKey),
		}
		dposV3Init, err := marshalInit(&dposv3.InitRequest{
			Params: &dposv3.Params{
				ValidatorCount:      21,
				ElectionCycleLength: 0,
				OracleAddress:       oracleAddr,
			},
			Validators: []*loom.Validator{
				validator,
			},
			InitCandidates: true,
		})
		if err != nil {
			return nil, err
		}

		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "dposV3",
			Location:   "dposV3:3.0.0",
			Init:       dposV3Init,
		})

	}

	//If this is enabled lets default to giving a genesis file with the plasma_cash contract
	if cfg.PlasmaCash.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "plasmacash",
				Location:   "plasmacash:1.0.0",
			})
	}

	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "ethcoin",
				Location:   "ethcoin:1.0.0",
			})
	}

	if cfg.AddressMapperContractEnabled() {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "addressmapper",
				Location:   "addressmapper:0.1.0",
			})
	}

	if cfg.SampleGoContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "sample-go-contract",
				Location:   "sample-go-contract:1.0.0",
			},
		)
	}

	if cfg.TransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "gateway",
				Location:   "gateway:0.1.0",
			})
	}

	if cfg.LoomCoinTransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "loomcoin-gateway",
				Location:   "loomcoin-gateway:0.1.0",
			})
	}

	if cfg.TronTransferGateway.ContractEnabled {
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "tron-gateway",
				Location:   "tron-gateway:0.1.0",
			})
	}

	contractOwner := &types.Address{
		ChainId: chainID,
		Local:   loom.LocalAddressFromPublicKey(validator.PubKey),
	}

	if cfg.BinanceTransferGateway.ContractEnabled {
		initBytes, err := marshalInit(&tgtypes.TransferGatewayInitRequest{
			Owner: contractOwner,
		})
		if err != nil {
			return nil, err
		}
		contracts = append(contracts,
			config.ContractConfig{
				VMTypeName: "plugin",
				Format:     "plugin",
				Name:       "binance-gateway",
				Location:   "binance-gateway:0.1.0",
				Init:       initBytes,
			},
		)
	}

	if cfg.ChainConfig.ContractEnabled {
		chainConfigInitRequest := cctypes.InitRequest{
			Owner: contractOwner,
			Features: []*cctypes.Feature{
				&cctypes.Feature{
					Name:   features.DPOSVersion3_1,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.ChainCfgVersion1_1,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.ChainCfgVersion1_2,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.ChainCfgVersion1_3,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.EvmTxReceiptsVersion2Feature,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.CoinVersion1_1Feature,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.AppStoreVersion3_1,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.AuthSigTxFeaturePrefix + "eth",
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.CheckTxValueFeature,
					Status: chainconfig.FeatureWaiting,
				},
				&cctypes.Feature{
					Name:   features.EvmConstantinopleFeature,
					Status: chainconfig.FeatureWaiting,
				},
			},
		}

		chainConfigInit, err := marshalInit(&chainConfigInitRequest)
		if err != nil {
			return nil, err
		}

		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "chainconfig",
			Location:   "chainconfig:1.0.0",
			Init:       chainConfigInit,
		})
	}

	if cfg.DeployerWhitelist.ContractEnabled {
		dwInitRequest := dwtypes.InitRequest{
			Owner: contractOwner,
		}

		dwInit, err := marshalInit(&dwInitRequest)
		if err != nil {
			return nil, err
		}

		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "deployerwhitelist",
			Location:   "deployerwhitelist:1.0.0",
			Init:       dwInit,
		})
	}

	if cfg.Karma.Enabled {
		karmaInitRequest := ktypes.KarmaInitRequest{
			Sources: []*ktypes.KarmaSourceReward{
				{Name: "example-award-token", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
			},
			Upkeep: &ktypes.KarmaUpkeepParams{
				Cost:   1,
				Period: 3600,
			},
			Config: &ktypes.KarmaConfig{MinKarmaToDeploy: karma.DefaultUpkeepCost},
		}
		oracle, err := loom.ParseAddress(cfg.Oracle)
		if err == nil {
			karmaInitRequest.Oracle = oracle.MarshalPB()
		}
		karmaInit, err := marshalInit(&karmaInitRequest)

		if err != nil {
			return nil, err
		}
		contracts = append(contracts, config.ContractConfig{
			VMTypeName: "plugin",
			Format:     "plugin",
			Name:       "karma",
			Location:   "karma:1.0.0",
			Init:       karmaInit,
		})
	}

	return &config.Genesis{
		Config:    *cconfig.DefaultConfig(),
		Contracts: contracts,
	}, nil
}
