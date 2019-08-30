
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
  // One index takes 256 bits.  
  // On this assertion 0x00 represent the first value that stored in StoreTestContract.
  // Which is storedUint1 and the value is '15'
    index = "0x00"
    result = await getStorageAt(ethUrl, storeContract.address, index)
    result = web3js.utils.hexToNumber(result)
    assert.equal(result, 15 ,"Invalid value at index "+index)

  //Because the type of storeUint2 and storedUint3 only take 128 + 32 bits to store.
  //It will be stored on the same index for storage optimization.
    index = "0x01"
    result = await getStorageAt(ethUrl, storeContract.address, index)
    assert.equal(result, '0x000000000000000000000000000004d20000000000000000000000000000429f', "Invalid value at index " + index)

  //Assertion of 'string1' in StoreTestContract
    index = "0x02"
    result = await getStorageAt(ethUrl, storeContract.address,index)
    result = await web3js.utils.hexToUtf8(result)
    assert.equal(result, 'test1', "Invalid value at index " + index)

  //Assertion of 'string2' in StoreTestContract
    index = "0x03"
    result = await getStorageAt(ethUrl, storeContract.address, index)
    result = await web3js.utils.hexToUtf8(result)
    assert.equal(result, "test1236", "Invalid value at index " + index)

  //Assertion of 'string3' in StoreTestContract
    index = "0x04"
    result = await getStorageAt(ethUrl, storeContract.address, index)
    result = await web3js.utils.hexToUtf8(result)
    assert.equal(result, "lets string something", "Invalid value at index " + index)

  //Assertion of 'uintarray' in StoreTestContract
  //This will have a result as 2 because at index 0x05 the contract store 2 value of uint as array.
    index = "0x05"
    result = await getStorageAt(ethUrl, storeContract.address, index)
    result = await web3js.utils.hexToNumber(result)
    assert.equal(result, 2, "Invalid value at index " + index)

    // uintarray[0]
    result = await getStorageAt(ethUrl, storeContract.address, "0x036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db0")
    result = await web3js.utils.hexToNumber(result)
    assert.equal(result, 8000, "Invalid value at index " + index)

    // uintarray[1]
    result = await getStorageAt(ethUrl, storeContract.address, "0x036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db1")
    result = await web3js.utils.hexToNumber(result)
    assert.equal(result, 9000, "Invalid value at index " + index)

  });

});
