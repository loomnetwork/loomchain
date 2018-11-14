const fs = require('fs');
const path = require('path');
const Web3 = require('web3');

contract('BlockNumber', async () => {
  let web3js;

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined')
    }
    const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim();
    web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));
  });


  it('eth_blockNumber', async () => {
      const blockNumber = await web3js.eth.getBlockNumber();
      assert(0 < blockNumber);
  })
});
