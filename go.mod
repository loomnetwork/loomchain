module github.com/loomnetwork/loomchain

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/allegro/bigcache v1.2.1-0.20190218064605-e24eb225f156
	github.com/btcsuite/btcd v0.0.0-20190109040709-5bda5314ca95
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/eosspark/eos-go v0.0.0-20190820132053-c626c17c72f9 // indirect
	github.com/eosspark/geos v0.0.0-20190820132053-c626c17c72f9 // indirect
	github.com/ethereum/go-ethereum v1.9.10
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/go-kit/kit v0.9.0
	github.com/gogo/protobuf v1.1.1
	github.com/golang/protobuf v1.3.3
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/go-plugin v1.0.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jmhodges/levigo v1.0.0
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/loomnetwork/gamechain v0.0.0-20200116065708-6b60967e1341
	github.com/loomnetwork/go-loom v0.0.0-20200213131121-2fc78efbd401
	github.com/loomnetwork/mamamerkle v0.0.0-20200206113614-cc12f6675a88
	github.com/miguelmota/go-solidity-sha3 v0.1.0
	github.com/phonkee/go-pubsub v0.0.0-20181130135233-5425e5981d13
	github.com/pkg/errors v0.9.1
	github.com/posener/wstest v0.0.0-20180216222922-04b166ca0bf1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/tendermint/btcd v0.1.0 // indirect
	github.com/tendermint/go-amino v0.14.0
	github.com/tendermint/iavl v0.0.0-00010101000000-000000000000
	github.com/tendermint/tendermint v0.0.0-00010101000000-000000000000
	github.com/ulule/limiter v2.2.1+incompatible
	golang.org/x/crypto v0.0.0-20200204104054-c9f3fb736b72
	golang.org/x/net v0.0.0-20190912160710-24e19bdeb0f2
	google.golang.org/genproto v0.0.0-20200205142000-a86caf926a67 // indirect
	google.golang.org/grpc v1.27.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.5 // indirect
)

replace github.com/phonkee/go-pubsub => github.com/loomnetwork/go-pubsub v0.0.0-20180626134536-2d1454660ed1

replace github.com/tendermint/iavl => github.com/loomnetwork/iavl v0.12.2-0.20190705112304-45ae3144d0e8

replace github.com/tendermint/tendermint => github.com/loomnetwork/tendermint v0.27.4-0.20191021121852-f32154d54e30

replace github.com/miguelmota/go-solidity-sha3 => github.com/loomnetwork/go-solidity-sha3 v0.0.2-0.20190227083338-45494d847b31

// Ensure nothing pulls in newer incompatible versions of the protobuf packages.
replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.2.1

replace github.com/prometheus/common => github.com/prometheus/common v0.7.0

replace github.com/golang/protobuf => github.com/golang/protobuf v1.1.0

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.1.1

replace google.golang.org/grpc => google.golang.org/grpc v1.20.1

// This is locked down to this particular revision because this is the last revision before the
// google.golang.org/genproto was recompiled with a new version of protoc, which produces pb.go files
// that don't appear to be compatible with the gogo protobuf & protoc versions we use.
// google.golang.org/genproto seems to be pulled in by the grpc package.
replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20190508193815-b515fa19cec8

// Don't remember why this is locked down anymore, something to do with some
// 'timeout waiting for connection info' error
replace github.com/hashicorp/go-plugin => github.com/hashicorp/go-plugin v0.0.0-20181211201406-f4c3476bd385

replace github.com/jmhodges/levigo => github.com/jmhodges/levigo v0.0.0-20161115193449-c42d9e0ca023

replace github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.0.0-20181130015935-7d2daa5bfef2

replace github.com/certusone/yubihsm-go => github.com/certusone/yubihsm-go v0.1.1-0.20190814054144-892fb9b370f3

// Temp workaround for https://github.com/prometheus/procfs/issues/221
replace github.com/prometheus/procfs => github.com/prometheus/procfs v0.0.6-0.20191003141728-d3b299e382e6

// prometheus/client_model is pulled by prometheus/client_golang so lock it down as well
replace github.com/prometheus/client_model => github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4

replace github.com/ethereum/go-ethereum => github.com/loomnetwork/go-ethereum v1.8.17-0.20200207100928-8e02782666c8
