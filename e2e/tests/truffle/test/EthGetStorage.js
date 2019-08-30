
const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const { waitForXBlocks, getStorageAt } = require('./helpers')
const StoreTestContract = artifacts.require('StoreTestContract');

// web3 functions called using truffle objects use the loomProvider
// web3 functions called uisng we3js access the loom QueryInterface directly
contract('StoreTestContract', async (accounts) => {

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined');
    }
    let nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8');
    web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));
    ethUrl = `http://${nodeAddr}/eth`
  });

  it('eth_getStorageAt', async () => {
    const storeContract = await StoreTestContract.deployed();

  // Parameters in contract storage are indexd from the beginning. 
  // One index takes 256 bytes.  
  // On this assertion 0x00 represent the first value that stored in StoreTestContract.
  // Which is storedUint1 and the value is '15'
    result = await getStorageAt(ethUrl, storeContract.address, "0x00")
    assert.equal(result, '0x000000000000000000000000000000000000000000000000000000000000000f')

  //Because the type of storeUint2 and storedUint3 only take 128 + 32 bytes to store.
  //It will be stored on the same index for storage optimization.
    result = await getStorageAt(ethUrl, storeContract.address, "0x01")
    assert.equal(result, '0x000000000000000000000000000004d20000000000000000000000000000429f')

  //Assertion of 'string1' in StoreTestContract
    result = await getStorageAt(ethUrl, storeContract.address,"0x02")
    result = web3js.utils.hexToUtf8(result)
    assert.equal(result, 'test1')

  //Assertion of 'string2' in StoreTestContract
    result = await getStorageAt(ethUrl, storeContract.address, "0x03")
    result = web3js.utils.hexToUtf8(result)
    assert.equal(result, "test1236", "invalid value get storage")

  //Assertion of 'string3' in StoreTestContract
    result = await getStorageAt(ethUrl, storeContract.address, "0x04")
    result = web3js.utils.hexToUtf8(result)
    assert.equal(result, "lets string something", "invalid value get storage")

  });

});
