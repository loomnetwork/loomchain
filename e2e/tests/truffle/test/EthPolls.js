const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const MyToken = artifacts.require('MyToken');
const util = require('ethereumjs-util');

contract('MyToken', async (accounts) => {
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

  it('eth_newBlockFilter', async () => {
  });

  it('eth_newPendingTransactionFilter', async () => {
  });

  it('eth_newFilter', async () => {
  });

});