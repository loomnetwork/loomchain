const Web3 = require('web3');
const fs = require('fs');
const path = require('path');
const EthereumTx = require('ethereumjs-tx').Transaction;
const { getLoomEvmTxHash, waitForXBlocks } = require('./helpers');
const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js');
const TxHashTestContract = artifacts.require('TxHashTestContract');

contract('debug_traceTransaction', async (accounts) => {
    let contract, fromAddr, nodeAddr, txHashTestContract, web3, web3js;
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

        txHashTestContract = await TxHashTestContract.deployed();
        contract = new web3.eth.Contract(TxHashTestContract._json.abi, txHashTestContract.address, {from});
    });

    it('Test debug_traceTransaction', async () => {
        const txResult = await contract.methods.set(1111).send();
        await web3js.currentProvider.send({
            method: "debug_traceTransaction",
            params: [txResult.transactionHash,{"disableStorage":false,"disableMemory":false,"disableStack":false}],
            jsonrpc: "2.0",
            id: new Date().getTime()
        }, function (error, result) {
            assert.equal(null, error, "debug_traceTransaction returned error");
            assert.equal(true, result.result.structLogs.length > 0, "trace did not return any data");
        });
        await waitForXBlocks(nodeAddr, 1)
    })
});