package dposv3

import "strings"

type DPOSConfig struct {
	BootstrapNodes           []string
	TotalStakedCacheDuration int64
}

func DefaultDPOSConfig() *DPOSConfig {
	return &DPOSConfig{
		BootstrapNodes: []string{
			"default:0x0e99fc16e32e568971908f2ce54b967a42663a26",
			"default:0xac3211caecc45940a6d2ba006ca465a647d8464f",
			"default:0x69c48768dbac492908161be787b7a5658192df35",
			"default:0x2a3a7c850586d4f80a12ac1952f88b1b69ef48e1",
			"default:0x4a1b8b15e50ce63cc6f65603ea79be09206cae70",
			"default:0x0ce7b61c97a6d5083356f115288f9266553e191e",
		},
		TotalStakedCacheDuration: 60, // 60 seconds
	}
}

func (dposCfg *DPOSConfig) BootstrapNodesList() map[string]bool {
	bootstrapNodesList := map[string]bool{}
	for _, addr := range dposCfg.BootstrapNodes {
		bootstrapNodesList[strings.ToLower(addr)] = true
	}
	return bootstrapNodesList
}
