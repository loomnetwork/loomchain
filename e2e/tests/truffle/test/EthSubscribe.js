const fs = require('fs');
const path = require('path');
const Web3 = require('web3');

contract('MyToken', async (accounts) => {
  let web3js;

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined');
    }
    let nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8');
    web3js = new Web3(new Web3.providers.WebsocketProvider(`ws://${nodeAddr}/eth`));

    alice = accounts[1];
    bob = accounts[2];
  });

  it('eth_subscribe', async () => {
      const sub1 = await web3js.eth.subscribe('newBlockHeaders');
      console.log("sub1", sub1)
  });

  it('eth_unsubscribe', async () => {
      var subscription = web3js.eth.subscribe('newBlockHeaders', function(error, result){
        console.log("\nresult", result);
        if (!error) {
              console.log(result);
              return;
          }
          console.error(error);
      })
      .on("\ndata", function(blockHeader){
          console.log(blockHeader);
      })
      .on("\nerror", console.error);

      console.log("\nsubscription",subscription);


      // unsubscribes the subscription
      subscription.unsubscribe(function(error, success){
          if (success) {
              console.log('\nSuccessfully unsubscribed!');
          } else {
            console.log("\nerror unscribing", error)
          }
      });
  });
});