const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const InnerEmitter = artifacts.require('InnerEmitter');
const OuterEmitter = artifacts.require('OuterEmitter');
const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, Contracts, Address
} = require('loom-js');

contract('SampleGoContract', async () => {
    const privateKey = CryptoUtils.generatePrivateKey();
    const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);
    let client, web3js;

    beforeEach(async () => {
        const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim();
        const chainID = 'default';
        const writeUrl = `ws://${nodeAddr}/websocket`;
        const readUrl = `ws://${nodeAddr}/queryws`;

        client = new Client(chainID, writeUrl, readUrl);
        client.on('error', msg => {
            console.error('Error on connect to client', msg);
            console.warn('Please verify if loom cluster is running')
        });

        const setupMiddlewareFn = function (client, privateKey) {
            const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);
            return [new SpeculativeNonceTxMiddleware(publicKey, client), new SignedTxMiddleware(privateKey)]
        };
        client.txMiddleware = setupMiddlewareFn(client, privateKey);

        web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));
    });

    it('nested event emitted from go contract', async () => {
        let innerEmitter = await InnerEmitter.deployed();
        let outerEmitter = await OuterEmitter.deployed();

        const innerEmitterValue = 31;
        const outerEmitterValue = 63;

        const sampleGoContract = await Contracts.SampleGoContract.createAsync(
            client,
            new Address(client.chainId, LocalAddress.fromPublicKey(publicKey))
        );

        await sampleGoContract.testNestedEvmCallsAsync(
            new Address(client.chainId, LocalAddress.fromHexString(innerEmitter.address)),
            new Address(client.chainId, LocalAddress.fromHexString(outerEmitter.address)),
            innerEmitterValue,
            outerEmitterValue,
        );
        const goContractLogs = await web3js.eth.getPastLogs({
            address: innerEmitter.address,
        });

        const receipt = await web3js.eth.getTransactionReceipt(goContractLogs[0].transactionHash);
        const logsFromGoContract = receipt.logs;

        assert.equal(2, logsFromGoContract.length, "number of logs from go contract");
        assert.equal(logsFromGoContract[0].topics[1], web3.utils.padLeft(innerEmitterValue, 64), "check inner emitter value");
        assert.equal(logsFromGoContract[1].topics[1], web3.utils.padLeft(outerEmitterValue, 64), "check outer emitter value");

        const innerEmitterResult = await innerEmitter.sendEvent(innerEmitterValue);
        const innerEmitterReceipt = await web3js.eth.getTransactionReceipt(innerEmitterResult.receipt.transactionHash);
        const logsFromInnerEmitter = innerEmitterReceipt.logs;

        assert.equal(1, logsFromInnerEmitter.length, "number of logs from InnerEmitter contract");
        assert.equal(2, logsFromInnerEmitter[0].topics.length, "number of topics" );
        assert.equal(logsFromGoContract[0].topics[0], logsFromInnerEmitter[0].topics[0], "function name topic");
        assert.equal(logsFromGoContract[0].topics[1], logsFromInnerEmitter[0].topics[1], "value topic");

        const outerEmitterResult = await outerEmitter.sendEvent(outerEmitterValue);
        const outerEmitterReceipt = await web3js.eth.getTransactionReceipt(outerEmitterResult.receipt.transactionHash);
        const logsFromOuterEmitter = outerEmitterReceipt.logs;

        assert.equal(1, logsFromOuterEmitter.length, "number of logs from OuterEmitter contract");
        assert.equal(2, logsFromOuterEmitter[0].topics.length, "number of topics" );
        assert.equal(logsFromGoContract[1].topics[0], logsFromOuterEmitter[0].topics[0], "function name topic");
        assert.equal(logsFromGoContract[1].topics[1], logsFromOuterEmitter[0].topics[1], "value topic");
    });

});