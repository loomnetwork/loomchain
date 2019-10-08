const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const ethers = require('ethers');
const TestLoomNativeApi = artifacts.require('TestLoomNativeApi');
const Accounts = require('web3-eth-accounts');
const {
  SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client, LoomProvider,
  LocalAddress, CryptoUtils, Contracts, Address, EthersSigner, getJsonRPCSignerAsync
} = require('loom-js');

contract('LoomNativeApi', async (accounts) => {
    const alice = accounts[1];
    const privateKey = CryptoUtils.generatePrivateKey();
    const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);
    const ethAccount = web3.eth.accounts.create();
    let client, web3js, web3lp;
    const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim();
    const httpProvider =  new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`)

    beforeEach(async () => {
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

    it('map loom account', async () => {
        const ethAddress = '0xffcf8fdee72ac11b5c542428b35eef5769c409f0';
        const from = new Address('eth', LocalAddress.fromHexString(ethAddress));
        const to = new Address(client.chainId, LocalAddress.fromPublicKey(publicKey));
        const jsonRpcProvider = new ethers.providers.JsonRpcProvider(`http://${nodeAddr}`);
        const signers = jsonRpcProvider.getSigner(ethAddress);
        const ethersSigner = new EthersSigner(signers);
        const addressMapper = await Contracts.AddressMapper.createAsync(
          client,
          new Address(client.chainId, LocalAddress.fromPublicKey(publicKey))
        );
        await addressMapper.addIdentityMappingAsync(from, to, ethersSigner);

        let testApi = await TestLoomNativeApi.deployed();
        var msg = '0x8CbaC5e4d803bE2A3A5cd3DbE7174504c6DD0c1C';
        var hash = web3js.utils.sha3(msg);
        var sig = await ethAccount.sign(hash);
        var result = await testApi.TestMappedLoomAccount.call("eth", hash, sig.v, sig.r, sig.s);
        assert.equal(result, ethAddress);
    });

    it('map any account', async () => {
        let testApi = await TestLoomNativeApi.deployed();
    })
});
