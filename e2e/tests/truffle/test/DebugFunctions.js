const Web3 = require('web3');
const fs = require('fs');
const path = require('path');
const EthereumTx = require('ethereumjs-tx').Transaction;
const { getLoomEvmTxHash } = require('./helpers');

const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js');

const TxHashTestContract = artifacts.require('TxHashTestContract');
const NonceTestContract = artifacts.require('NonceTestContract');
const SimpleStore = artifacts.require('SimpleStore');

// Requires receipts:v3.3 to be enabled, and receipts:v3.4 not to be, but the new tx hash algo needs
// more review & testing before we can release it so skipping this test for now.
contract('TxHashTestContract', async (accounts) => {
    let contract, nonceContract, fromAddr, nodeAddr, txHashTestContract, nonceTestContract, web3, web3js;

    beforeEach(async () => {
        nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        const chainID = 'default';
        const writeUrl = `ws://${nodeAddr}/websocket`;
        const readUrl = `ws://${nodeAddr}/queryws`;

        const privateKey = CryptoUtils.generatePrivateKey();
        const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);

        fromAddr = LocalAddress.fromPublicKey(publicKey);
        const from = fromAddr.toString();

        var client = new Client(chainID, writeUrl, readUrl);
        client.on('error', msg => {
            console.error('Error on connect to client', msg);
            console.warn('Please verify if loom cluster is running')
        });
        const setupMiddlewareFn = function(client, privateKey) {
            const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);
            return [new SpeculativeNonceTxMiddleware(publicKey, client), new SignedTxMiddleware(privateKey)]
        };
        var loomProvider = new LoomProvider(client, privateKey, setupMiddlewareFn);

        web3 = new Web3(loomProvider);
        web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));


        //nonceTestContract = await SimpleStore.deployed();
        //nonceContract = new web3.eth.Contract(SimpleStore._json.abi, nonceTestContract.address, {from});

        txHashTestContract = await TxHashTestContract.deployed();
        contract = new web3.eth.Contract(TxHashTestContract._json.abi, txHashTestContract.address, {from});
    });

    it('Test debug_traceTransaction', async () => {
        console.log("piers contract.methods|", contract.methods);
        console.log("piers contract.method.set|", contract.methods.set);

       // console.log("piers nonceTestContract.methods|", nonceTestContract.methods);
       // console.log("piers nonceTestContract.methods.setValue|", nonceTestContract.methods.setValue);
       // console.log("piers nonceTestContract.methods.setValue|", nonceTestContract.methods.setValue(uint256));
        try {
/*
            const nonceTxResult = await nonceTestContract.methods.set(1111).send();
            console.log("piers nonceTxResult txResult.tx|", nonceTxResult.tx);
            console.log("piers nonceTestContract txResult.transactionHash|", nonceTxResult.transactionHash);
            await web3js.currentProvider.send({
                method: "debug_traceTransaction",
                params: [nonceTxResult.transactionHash],
                jsonrpc: "2.0",
                id: new Date().getTime()
            }, function (error, result) {
                console.log("piers!!!!!!!!!nonceTxResult debug_traceTransaction sendResult|", result, "error", error)
                console.log("piersn onceTxResult  failed|", result.failed);
                console.log("piers nonceTxResult structLogs|", result.structLogs);
                //assert.equal(true, result === result);
                assert.equal(false, result.failed)
            });
*/
            const txResult = await contract.methods.set(1111).send();
            console.log("piers txResult.tx|", txResult.tx);
            console.log("piers txResult.transactionHash|", txResult.transactionHash);

            await web3js.currentProvider.send({
                method: "debug_traceTransaction",
                params: [txResult.transactionHash,{"disableStorage":true,"disableMemory":false,"disableStack":false,"fullStorage":false}],
                jsonrpc: "2.0",
                id: new Date().getTime()
            }, function (error, result) {
                console.log("piers!!!!!!!!! debug_traceTransaction sendResult|", result, "error", error)
                console.log("piers failed|", result.failed);
                console.log("piers structLogs|", result.structLogs);
                //assert.equal(true, result === result);
                assert.equal(undefined, result.failed)
            });
            //console.log("sendResult3", sendResult3);

            //const doSomething = async () => {
            //    await sleep2(2000)
            //    //do stuff
            //};
            //console.log("after sleep")

            const sendResult = await web3js.currentProvider.send({
                method: "eth_blockNumber",
                params: [],
                jsonrpc: "2.0",
                id: new Date().getTime()
            }, function (error, result) {
                console.log("send eth_blockNumber|", result, "error", error)
            });
            //console.log("sendResult", sendResult);


            const receipt = await web3js.eth.getTransactionReceipt(txResult.transactionHash);
            //const receipt = await web3js.eth.getTransactionReceipt(txResult.tx);
            console.log("piers receipt|", receipt);

            //sleep(1000000)



        } catch(err) {
            console.log("caught error", err);
        }
        //const dttResult =  await web3js.currentProvider.send('debug_traceTransaction', [ txResult.transactionHash ]).then((result) => {
        //    console.log(result);
        //});
        //console.log("dttResult", dttResult)
    })

});

function sleep1(milliseconds) {
    let timeStart = new Date().getTime();
    while (true) {
        let elapsedTime = new Date().getTime() - timeStart;
        if (elapsedTime > milliseconds) {
            break;
        }
    }
}

const sleep = (milliseconds) => {
    return new Promise(resolve => setTimeout(resolve, milliseconds))
}