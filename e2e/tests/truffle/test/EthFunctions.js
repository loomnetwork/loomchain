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

  it('eth_blockNumber', async () => {
      const blockNumber = await web3js.eth.getBlockNumber();
      assert(0 < blockNumber);
  });

  it('getPastLogs', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(97, { from: alice });
    await tokenContract.mintToken(98, { from: alice });
    await tokenContract.mintToken(99, { from: bob });

    const myTokenLogs = await web3js.eth.getPastLogs({
      address: tokenContract.address
    });
    for (i=0 ; i<myTokenLogs.length ; i++) {
      assert.equal(myTokenLogs[i].address.toLowerCase(), tokenContract.address)
    }

    const aliceLogs = await web3js.eth.getPastLogs({
      topics: [null, null, web3js.utils.padLeft(alice, 64), null]
    });
    for (i=0 ; i<aliceLogs.length ; i++) {
      assert.equal(aliceLogs[i].topics[2], web3js.utils.padLeft(alice, 64))
    }
  });

  it('eth_getTransactionReceipt', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(101, { from: alice });
    assert.equal(tokenContract.address, result.receipt.contractAddress);

    const receipt = await web3js.eth.getTransactionReceipt(result.tx);
    assert.equal(receipt.to, result.receipt.contractAddress);
    assert.equal(receipt.from, alice);
    assert.equal(1, receipt.logs.length);
    assert.equal(4, receipt.logs[0].topics.length);
    assert.equal(alice, receipt.logs[0].address.toLowerCase());
  });

  it('eth_getTransactionByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(102, { from: alice });
    const txObj = await web3js.eth.getTransaction(result.tx);

    assert.equal(txObj.to.toLowerCase(), result.receipt.contractAddress);
    assert.equal(txObj.from.toLowerCase(), alice);
  });

  it('eth_getCode', async () => {
    const tokenContract = await MyToken.deployed();
    const code = await web3js.eth.getCode(tokenContract.address);
    assert.equal(tokenContract.constructor._json.deployedBytecode, code)
  });

  it('eth_getBlockByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(103, { from: alice });
    await tokenContract.mintToken(104, { from: alice });
    const txObject = await web3js.eth.getTransaction(result.tx, true);

    const blockByHash = await web3js.eth.getBlock(txObject.blockHash, true);
    assert.equal(txObject.blockHash, blockByHash.hash);
    assert.equal(result.receipt.blockNumber, blockByHash.number);

    assert.equal(1, blockByHash.transactions.length);
    assert.equal(alice , blockByHash.transactions[0].from.toLowerCase());
    assert.equal(tokenContract.address ,blockByHash.transactions[0].to.toLowerCase());
    assert.equal(result.receipt.blockNumber ,blockByHash.transactions[0].blockNumber);
    assert.equal(result.tx ,blockByHash.transactions[0].hash);
    assert.equal(txObject.blockHash ,blockByHash.transactions[0].blockHash);

    const blockByHashFalse = await web3js.eth.getBlock(txObject.blockHash, false);
    assert.equal(result.tx, blockByHashFalse.transactions[0]);
  });

  it('eth_getBlockTransactionCountByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(105, { from: alice });
    // Do second transaction to move to next block
    await tokenContract.mintToken(106, { from: alice });
    const txObject = await web3js.eth.getTransaction(result.tx, true);

    const txCount = await web3js.eth.getBlockTransactionCount(txObject.blockHash);
    assert.equal(txCount, 1);
  });

  it('eth_getTransactionByBlockHashAndIndex', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(107, { from: alice });
    // Do second transaction to move to next block
    await tokenContract.mintToken(108, { from: alice });
    const txObject = await web3js.eth.getTransaction(result.tx, true);

    const txObj = await web3js.eth.getTransactionFromBlock(txObject.blockHash, 0);
    assert.equal(alice , txObj.from.toLowerCase());
    assert.equal(tokenContract.address ,txObj.to.toLowerCase());
    assert.equal(result.receipt.blockNumber ,txObj.blockNumber);
    assert.equal(result.tx ,txObj.hash);
    assert.equal(txObject.blockHash, txObj.blockHash);
  });

  it('eth_Call', async () => {
    const tokenContract = await MyToken.deployed();
    await tokenContract.mintToken(112, { from: alice });

    let owner = await tokenContract.ownerOf.call(112);
    const ethOwner = await web3js.eth.call({
      to: tokenContract.address,
      data: "0x6352211e0000000000000000000000000000000000000000000000000000000000000070" // abi for ownerOf(12)
    },"latest");
    assert.equal(ethOwner, web3js.utils.padLeft(owner, 64));
  });

});

