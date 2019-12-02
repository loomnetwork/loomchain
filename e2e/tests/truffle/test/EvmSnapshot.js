const {
    waitForXBlocks,
    ethGetTransactionCount
} = require('./helpers')
const Web3 = require('web3')
const fs = require('fs')
const path = require('path')
const {
    SpeculativeNonceTxMiddleware,
    SignedTxMiddleware,
    Client,
    EthersSigner,
    createDefaultTxMiddleware,
    Address,
    LocalAddress,
    CryptoUtils,
    LoomProvider,
    Contracts
} = require('loom-js')
const ethers = require('ethers').ethers

const NonceTestContract = artifacts.require('NonceTestContract');

// web3 functions called using truffle objects use the loomProvider
// web3 functions called uisng we3js access the loom QueryInterface directly
contract('TestEvmSnapshot', async (accounts) => {
    // This test is not provider dependent so just run it with Loom Truffle provider
    if (process.env.TRUFFLE_PROVIDER === 'hdwallet') {
        return
    }

    let contract, from, nodeAddr

    beforeEach(async () => {
        nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()

        const client = new Client('default', `ws://${nodeAddr}/websocket`, `ws://${nodeAddr}/queryws`)
        client.on('error', msg => {
            console.error('Error on connect to client', msg)
            console.warn('Please verify if loom cluster is running')
        })
        const privKey = CryptoUtils.generatePrivateKey()
        const pubKey = CryptoUtils.publicKeyFromPrivateKey(privKey)
        client.txMiddleware = createDefaultTxMiddleware(client, privKey);

        const setupMiddlewareFn = function (client, privateKey) {
            const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)
            return [new SpeculativeNonceTxMiddleware(publicKey, client), new SignedTxMiddleware(privateKey)]
        }
        const loomProvider = new LoomProvider(client, privKey, setupMiddlewareFn)
        const web3 = new Web3(loomProvider)

        // Create a mapping between a DAppChain account & an Ethereum account so that
        // ethGetTransactionCount can resolve the Ethereum address it's given to a DAppChain address
        const localAddr = new Address(client.chainId, LocalAddress.fromPublicKey(pubKey));
        const addressMapper = await Contracts.AddressMapper.createAsync(client, localAddr);
        const ethAccount = web3.eth.accounts.create();
        const ethWallet = new ethers.Wallet(ethAccount.privateKey);
        await addressMapper.addIdentityMappingAsync(
            localAddr,
            new Address('eth', LocalAddress.fromHexString(ethAccount.address)),
            new EthersSigner(ethWallet)
        );
        from = ethAccount.address

        const nonceTestContract = await NonceTestContract.deployed()
        contract = new web3.eth.Contract(
            NonceTestContract._json.abi,
            nonceTestContract.address,
            // contract calls go through LoomProvider, which expect the sender address to be
            // a local address (not an eth address)
            {
                from: localAddr.local.toString()
            }
        );
    })
    it('SnapshotTest', async () => {
        for (var i = 0; i < 50; i++) {
            contract.methods.set(7777).send().then()
            contract.methods.get().call().then()
        }

        for (var i = 0; i < 50; i++) {
            contract.methods.set(8888).send().then()
            contract.methods.get().call().then()
        }
        await waitForXBlocks(nodeAddr, 5)
        await contract.methods.set(9999).send().then()
        assert.equal(await contract.methods.get().call(), 9999)
    });

});