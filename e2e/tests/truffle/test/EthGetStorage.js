
const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const { getStorageAt } = require('./helpers')
const StoreTestContract = artifacts.require('StoreTestContract');


contract('StoreTestContract', async (accounts) => {

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined');
    }
    let nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8');
    web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));
    ethUrl = `http://${nodeAddr}/eth`
    alice = accounts[1];
    bob = accounts[2];
  });

  it('eth_getStorageAt', async () => {
    const storeContract = await StoreTestContract.deployed();

    result = await getStorageAt(ethUrl, storeContract.address, "0x00")
    assert.equal(result, '0x000000000000000000000000000000000000000000000000000000000000000f')

    result = await getStorageAt(ethUrl, storeContract.address, "0x01")
    assert.equal(result, '0x000000000000000000000000000004d20000000000000000000000000000429f')

    result = await getStorageAt(ethUrl, storeContract.address,"0x02")
    result = web3js.utils.hexToUtf8(result)
    assert.equal(result, 'test1')

    result = await getStorageAt(ethUrl, storeContract.address, "0x03")
    result = web3js.utils.hexToUtf8(result)
    assert.equal(result, "test1236", "invalid value get storage")

  });

});
