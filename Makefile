PKG = github.com/loomnetwork/loom
GOFLAGS = -tags "evm"
PROTOC = protoc --plugin=./protoc-gen-gogo -Ivendor -I$(GOPATH)/src -I/usr/local/include
PLUGIN_DIR = $(GOPATH)/src/github.com/loomnetwork/go-loom

.PHONY: all clean test install deps proto

all: loom ladmin

loom ladmin: proto
	go build $(GOFLAGS) $(PKG)/cmd/$@

install: proto
	go install $(GOFLAGS) $(PKG)/cmd/loom $(PKG)/cmd/ladmin

protoc-gen-gogo:
	go build github.com/gogo/protobuf/protoc-gen-gogo

%.pb.go: %.proto protoc-gen-gogo
	$(PROTOC) --gogo_out=$(GOPATH)/src $(PKG)/$<

proto: vm/vm.pb.go

$(PLUGIN_DIR):
	git clone -q git@github.com:loomnetwork/go-loom.git $@

deps: $(PLUGIN_DIR)
	go get \
		golang.org/x/crypto/ed25519 \
		google.golang.org/grpc \
		github.com/gogo/protobuf/gogoproto \
		github.com/gogo/protobuf/proto \
		github.com/hashicorp/go-plugin \
		github.com/spf13/cobra \
		github.com/ethereum/go-ethereum
	dep ensure -vendor-only

test: proto
	go test $(GOFLAGS) $(PKG)/...

clean:
	go clean
	rm -f \
		loom \
		ladmin \
		protoc-gen-gogo \
		vm/vm.pb.go
