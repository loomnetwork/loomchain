module github.com/loomnetwork/loomchain

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/allegro/bigcache v1.1.0 // indirect
	github.com/btcsuite/btcd v0.0.0-20181123190223-3dcf298fed2d // indirect
	github.com/btcsuite/btcutil v0.0.0-20180706230648-ab6388e0c60a // indirect
	github.com/certusone/yubihsm-go v0.1.0
	github.com/cevaris/ordered_map v0.0.0-20180310183325-0efaee1733e3 // indirect
	github.com/ethereum/go-ethereum v1.8.19
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/go-kit/kit v0.8.0
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/gogo/protobuf v1.1.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/go-plugin v0.0.0-20181030172320-54b6ff97d818
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jmhodges/levigo v0.0.0-20161115193449-c42d9e0ca023 // indirect
	github.com/jtolds/gls v4.2.1+incompatible // indirect
	github.com/karalabe/hid v0.0.0-20181128192157-d815e0c1a2e2 // indirect
	github.com/loomnetwork/go-loom v0.0.0-20181128064951-262cc282f909
	github.com/loomnetwork/mamamerkle v0.0.0-20180929134451-bd379c19d963
	github.com/magiconair/properties v1.8.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/miguelmota/go-solidity-sha3 v0.0.0-20180712214648-92fbf5a798e8
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/onsi/ginkgo v1.7.0 // indirect
	github.com/onsi/gomega v1.4.3 // indirect
	github.com/pelletier/go-toml v1.2.0 // indirect
	github.com/perlin-network/life v0.0.0-20181118045116-6bf6615afaa9
	github.com/phonkee/go-pubsub v0.0.0-20180608135955-32036bad41e3
	github.com/pkg/errors v0.8.0
	github.com/prometheus/client_golang v0.9.1
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910 // indirect
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275 // indirect
	github.com/prometheus/procfs v0.0.0-20181129180645-aa55a523dc0a // indirect
	github.com/rcrowley/go-metrics v0.0.0-20180503174638-e2704e165165 // indirect
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d // indirect
	github.com/smartystreets/goconvey v0.0.0-20181108003508-044398e4856c // indirect
	github.com/spf13/afero v1.1.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/jwalterweatherman v1.0.0 // indirect
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.0.3
	github.com/stretchr/testify v1.2.2
	github.com/syndtr/goleveldb v0.0.0-20181105012736-f9080354173f
	github.com/tendermint/btcd v0.0.0-20180816174608-e5840949ff4f // indirect
	github.com/tendermint/go-amino v0.14.0
	github.com/tendermint/iavl v0.11.1
	github.com/tendermint/tendermint v0.26.3
	github.com/ulule/limiter v2.2.1+incompatible
	golang.org/x/crypto v0.0.0-20180830192347-182538f80094
	golang.org/x/net v0.0.0-20181011144130-49bb7cea24b1
	google.golang.org/grpc v1.16.0
	gopkg.in/yaml.v2 v2.2.1
)

replace github.com/loomnetwork/go-loom => ../go-loom

replace github.com/phonkee/go-pubsub => github.com/loomnetwork/go-pubsub v0.0.0-20180626134536-2d1454660ed1

replace github.com/perlin-network/life => ../life

replace github.com/go-interpreter/wagon => github.com/perlin-network/wagon v0.3.1-0.20180825141017-f8cb99b55a39
