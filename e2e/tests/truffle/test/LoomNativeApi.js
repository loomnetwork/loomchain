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
    const alice = accounts[1]
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

        var loomProvider = new LoomProvider(client, privateKey, setupMiddlewareFn)
        web3lp = new Web3(loomProvider)
    });

    it('map loom account', async () => {
        const addressMapper = await Contracts.AddressMapper.createAsync(
            client,
            new Address(client.chainId, LocalAddress.fromPublicKey(publicKey))
        );

        const ethAddress = '0xffcf8fdee72ac11b5c542428b35eef5769c409f0'
        const from = new Address('eth', LocalAddress.fromHexString(ethAddress));
        const to = new Address(client.chainId, LocalAddress.fromPublicKey(publicKey));

        console.log("piers node address", `http://${nodeAddr}`);
        const jsonRpcProvider = new ethers.providers.JsonRpcProvider(`http://${nodeAddr}`);
        const signers = jsonRpcProvider.getSigner(alice);

        const ethersSigner = new EthersSigner(signers);
       // console.log("piers ethersSigner",ethersSigner,"\npiers end ethersSigner");
        //console.log("piers from address", from);
        //console.log("piers to address", to);
        //await addressMapper.addIdentityMappingAsync(from, to, ethersSigner);
        //const testMapping = await addressMapper.getMappingAsync(from);
        //t.assert(from.equals(testMapping.from), 'Identity "from" correctly returned');
        //t.assert(to.equals(testMapping.to), 'Identity "to" correctly returned');

        let testApi = await TestLoomNativeApi.deployed();

        var msg = '0x8CbaC5e4d803bE2A3A5cd3DbE7174504c6DD0c1C';
        var hash = web3js.utils.sha3(msg);
        console.log("piers sha msg", msg);
        console.log("piers sha hash", hash);
        //var sig = web3js.eth.sign(alice, h);
        //var sig = web3js.eth.sign(ethAddress, h); //0x4d55361a8f362c8fc244dbd1e4968ca4b96d58e63a0f0c01a2cad1149106568a
        //var sig = await web3lp.eth.sign(h, "0x135a7de83802408321b74c322f8558db1679ac20");
        //var sig = await web3lp.eth.sign(h, ethAccount.address);
        var sig = await ethAccount.sign(hash);
        console.log("piers sig", sig);
        console.log("piers v ",sig.v, " r " ,sig.r," s ",sig.s);

        //sig = sig.slice(2);

       // var r = `0x${sig.slice(0, 64)}`;
        //var s = `0x${sig.slice(64, 128)}`;
        var result = await testApi.TestMappedAccount.call("eth", hash, sig.v, sig.r, sig.s);
        console.log("piers result TestMappedAccount", result);
        console.log("piers result alice", alice);
        //assert.equal(result, alice);

    });

    it('map any account', async () => {
        let testApi = await TestLoomNativeApi.deployed();
    })
});
