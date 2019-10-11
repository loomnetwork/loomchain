const rp = require('request-promise')
const keccak256 = require('js-sha3').keccak256
const web3 = require('web3')

async function assertRevert(promise) {
  try {
    await promise
    assert.fail('Expected revert not received')
  } catch (error) {
    const revertFound = error.message.search('revert') >= 0
    assert(revertFound, `Expected "revert", got ${error} instead`)
  }
}

async function delay(delayInms) {
  return new Promise(resolve  => {
    setTimeout(() => {
      resolve();
    }, delayInms);
  });
}

async function waitForXBlocks(nodeAddr, block) {
  block = Number(block)
  const ethUrl = `http://${nodeAddr}/rpc/status`
  var options = {
      method: 'GET',
      uri: ethUrl,
      json: true
  };
  const res = await rp(options)
  const currentBlock = Number(res.result.sync_info.latest_block_height)
  console.log("Current block", currentBlock)
  var retry = 60
  for(var i=0; i<retry;i++ ){
    await delay(1000)
    const res = await rp(options)
    var latestBlock = Number(res.result.sync_info.latest_block_height);
    console.log("Latest block", latestBlock)
    if (latestBlock >= currentBlock + block) {
      break;
    }
  }
  return
}

function getEventSignature(contract, eventName) {
  const eventJsonInterface = web3.utils._.find(
    contract._jsonInterface,
    o => o.name === eventName && o.type === 'event',
  )
  return eventJsonInterface.signature
}

/**
 * Returns the JSON interface of the given contract method.
 * @param {web3.eth.Contract} contract Contract instance
 * @param {string} funcName Contract method name
 */
function getContractFuncInterface(contract, funcName) {
  const jsonInterface = web3.utils._.find(
    contract._jsonInterface,
    o => o.name === funcName && o.type === 'function',
  )
  return jsonInterface
}

async function getNonce(nodeAddr, account) {
  const ethUrl = `http://${nodeAddr}/eth`
  var options = {
      method: 'POST',
      uri: ethUrl,
      body: {
          jsonrpc: '2.0',
          method: 'eth_getTransactionCount',
          params: [account, "latest"],
          id: 83,
      },
      json: true 
  };
  
  const res = await rp(options)
  return res.result
}

async function getStorageAt(ethUrl,account,position,block){
  var options = {
    method: 'POST',
    uri: ethUrl,
    body: {
      jsonrpc: '2.0',
      method: 'eth_getStorageAt',
      params: [account, position, block],
      id: 83,
    },
    json: true
  };

  const res = await rp(options)
  return res.result
}

async function getLatestBlock(nodeAddr) {
  const ethUrl = `http://${nodeAddr}/rpc/status`
  var options = {
    method: 'GET',
    uri: ethUrl,
    json: true
  };
  const res = await rp(options)
  return currentBlock = Number(res.result.sync_info.latest_block_height)
}

/**
 * Generates a hash for an EVM tx that will be executed by a Loom node.
 * This hash can be used to lookup the corresponding tx receipt.
 * @param {EthereumTx} ethTx Unsigned Ethereum transaction.
 * @param {Web3Address} fromAddr Sender address.
 */
function getLoomEvmTxHash(ethTx, fromAddr) {
  return keccak256(Buffer.concat([
    Buffer.from(ethTx.hash()),
    Buffer.from(fromAddr.bytes)
  ])).toString('hex')
}

module.exports = {
  assertRevert, delay, waitForXBlocks, getNonce, getStorageAt, 
  getLatestBlock, getLoomEvmTxHash, getEventSignature, getContractFuncInterface
}
