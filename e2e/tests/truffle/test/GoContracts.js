const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const TestEvent = artifacts.require('TestEvent');
const ChainTestEvent = artifacts.require('ChainTestEvent');
const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider, Contracts, Address
} = require('loom-js');




contract('SampleGoContract', async () => {
    const privateKey = CryptoUtils.generatePrivateKey();
    const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);
    let client, web3js;
    let testEventContract, chainTestEventContract;

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
        let loomProvider = new LoomProvider(client, privateKey, setupMiddlewareFn);
        client.txMiddleware = setupMiddlewareFn(client, privateKey);


        web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));

        let web3 = new Web3(loomProvider);
        testEventContract = await TestEvent.deployed();
        chainTestEventContract = await ChainTestEvent.deployed();

    });

    it('nested call from go contract', async () => {
        const sampleGoContract = await new Contracts.SampleGoContract.createAsync(
            client,
            new Address(client.chainId, LocalAddress.fromPublicKey(publicKey))
        );

        const goResult = await sampleGoContract.testNestedEvmCalls2Async(
            new Address(client.chainId, LocalAddress.fromHexString(testEventContract.address)),
            new Address(client.chainId, LocalAddress.fromHexString(chainTestEventContract.address))
        );
        const goContractLogs = await web3js.eth.getPastLogs({
            address: testEventContract.address,
        });

        const receipt = await web3js.eth.getTransactionReceipt(goContractLogs[0].transactionHash);
        const logsFromGoContract = receipt.logs;
        assert.equal(2, logsFromGoContract.length, "number of logs from go contract");

        const testEventResult = await testEventContract.sendEvent(65);
        const testEventReceipt = await web3js.eth.getTransactionReceipt(testEventResult.receipt.transactionHash);

        const logsFromTestEvent = testEventReceipt.logs;
        assert.equal(1, logsFromTestEvent.length, "number of logs from TestEvent contract");
        assert.equal(2, logsFromTestEvent[0].topics.length, "number of topics" );
        assert.equal(logsFromGoContract[0].topics[0], logsFromTestEvent[0].topics[0], "function name topic");
        assert.equal(logsFromGoContract[0].topics[1], logsFromTestEvent[0].topics[1], "value topic");

        const chainTestEventResult = await chainTestEventContract.chainEvent(33);
        const chainTestEventReceipt = await web3js.eth.getTransactionReceipt(chainTestEventResult.receipt.transactionHash);

        const logsFromChainTestEvent = chainTestEventReceipt.logs;
        assert.equal(1, logsFromChainTestEvent.length, "number of logs from ChainTestEvent contract");
        assert.equal(2, logsFromChainTestEvent[0].topics.length, "number of topics" );
        assert.equal(logsFromGoContract[1].topics[0], logsFromChainTestEvent[0].topics[0], "function name topic");
        assert.equal(logsFromGoContract[1].topics[1], logsFromChainTestEvent[0].topics[1], "value topic");
    });

});