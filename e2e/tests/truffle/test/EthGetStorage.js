
const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
//const MySimpleStore = artifacts.require('SimpleStorage');
const MyCoin = artifacts.require('MyCoin');

// web3 functions called using truffle objects use the loomProvider
// web3 functions called uisng we3js access the loom QueryInterface directly
contract('MyCoin', async (accounts) => {
  let web3js;

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined');
    }
    let nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8');
    web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));

    alice = accounts[1];
    bob = accounts[2];
  });

  it('eth_blockNumber', async () => {
      const blockNumber = await web3js.eth.getBlockNumber();
      console.log(blockNumber);
      assert(0 < blockNumber);
  });

});

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}
