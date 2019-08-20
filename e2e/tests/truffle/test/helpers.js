const rp = require('request-promise')

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
      console.log("break")
      break;
    }
  }
  return
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

module.exports = { assertRevert, delay, waitForXBlocks, getNonce}