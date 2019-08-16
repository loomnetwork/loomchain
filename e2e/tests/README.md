`cluster.sh` can be used to spin up a local 4-node cluster for testing.


## Setup

```bash
# in repo root...
make loom-gateway
make validators-tool
export LOOM_BIN=`pwd`/loom
export LOOM_VALIDATORS_TOOL=`pwd`/e2e/validators-tool

# setup truffle
cd e2e/tests/truffle
yarn
```


## Testing

```bash
cd e2e/tests/receipts
./run_truffle_tests.sh
```
