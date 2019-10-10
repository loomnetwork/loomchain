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
    let web3js, wallet, loomAddress;
    let testApi, testHash, sig;
    const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim();
    const msg = '0x8CbaC5e4d803bE2A3A5cd3DbE7174504c6DD0c1C';

    beforeEach(async () => {
        const chainID = 'default';
        const writeUrl = `ws://${nodeAddr}/websocket`;
        const readUrl = `ws://${nodeAddr}/queryws`;

        const privateKey = CryptoUtils.generatePrivateKey();
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

        publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey);
        loomAddress = new Address(client.chainId, LocalAddress.fromPublicKey(publicKey));
        wallet = ethers.Wallet.createRandom();
        ethAddress = await wallet.getAddress();

        const addressMapper = await Contracts.AddressMapper.createAsync(
          client,
          loomAddress
        );
        const from = new Address('eth', LocalAddress.fromHexString(ethAddress));
        await addressMapper.addIdentityMappingAsync(from, loomAddress,  new EthersSigner(wallet));

        testApi = await TestLoomNativeApi.deployed();
        testHash = web3js.utils.sha3(msg);
        sig = await wallet.signMessage(testHash).then(result => ethers.utils.splitSignature(result));
    });

    it('map loom account', async () => {
        const mappedAddress = await testApi.TestMappedLoomAccount.call('eth', testHash, sig.v, sig.r, sig.s);
        assert.equal(loomAddress.local.toString().toLowerCase(), mappedAddress.toLowerCase());
    });

    it('map any account', async () => {
      const mappedAddress = await testApi.TestMappedAccount.call('eth', testHash, sig.v, sig.r, sig.s, 'default');
      assert.equal(loomAddress.local.toString().toLowerCase(), mappedAddress.toLowerCase());
    })
});
