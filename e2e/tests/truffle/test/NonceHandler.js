const { waitForXBlocks, getNonce } = require('./helpers')
const Web3 = require('web3')
const fs = require('fs')
const path = require('path')

 const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js')

 const NonceTestContract = artifacts.require('NonceTestContract')

 contract('NonceTestContract', async (accounts) => {
    let contract, from, nodeAddr

    beforeEach(async () => {
        nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        const chainID = 'default'
        const writeUrl = `ws://${nodeAddr}/websocket`
        const readUrl = `ws://${nodeAddr}/queryws`

        const privateKey = CryptoUtils.generatePrivateKey()
        const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)

        from = LocalAddress.fromPublicKey(publicKey).toString()

        var client = new Client(chainID, writeUrl, readUrl)
        client.on('error', msg => {
            console.error('Error on connect to client', msg)
            console.warn('Please verify if loom cluster is running')
        })
        const setupMiddlewareFn = function(client, privateKey) {
          const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)
          return [new SpeculativeNonceTxMiddleware(publicKey, client), new SignedTxMiddleware(privateKey)]
        }
        var loomProvider = new LoomProvider(client, privateKey, setupMiddlewareFn)

        let web3 = new Web3(loomProvider)
        let nonceTestContract = await NonceTestContract.deployed()
        contract = new web3.eth.Contract(NonceTestContract._json.abi, nonceTestContract.address, {from});
    })

     it('Test nonce handler with failed txs', async () => {
        // send three reverted txs
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            assert(err)
        }
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            assert(err)
        }
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            assert(err)
        }

        await waitForXBlocks(nodeAddr, 2)
        let nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment even if the txs reverted
        assert.equal("0x3", nonce)

         // send three more reverted txs without await
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.err().send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 2)
        nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment even if the txs reverted
        assert.equal("0x6", nonce)
    })

    it('Test nonce handler with successful txs', async () => {
        // send three successful txs
        try {
            await contract.methods.set(1111).send()
            await contract.methods.set(2222).send()
            await contract.methods.set(3333).send()
        } catch(err) {
            assert.fail("transaction reverted");
        }
        
        await waitForXBlocks(nodeAddr, 2)
        let nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment 
        assert.equal("0x3", nonce)

        // send three more successful txs without await
        contract.methods.set(4444).send().then().catch(function(e) {})
        contract.methods.set(5555).send().then().catch(function(e) {})
        contract.methods.set(6666).send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 2)
        nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment
        assert.equal("0x6", nonce)
    })

    it('Test nonce handler with mixed txs', async () => {
        // send a mix of successful & failed txs
        try {
            await contract.methods.set(1111).send()
        } catch(err) {
            assert.fail("transaction reverted: " + err);
        }
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            assert(err)
        }
        try {
            await contract.methods.set(2222).send()
        }catch(err) {
            assert.fail("transaction reverted: " + err);
        }
        
        
        await waitForXBlocks(nodeAddr, 2)
        let nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment 
        assert.equal("0x3", nonce)

        // send three more mixed txs without await
        contract.methods.set(4444).send().then().catch(function(e) {
            assert.fail("transaction reverted: " + err);
        })
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.set(6666).send().then().catch(function(e) {
            assert.fail("transaction reverted: " + err);
        })

        await waitForXBlocks(nodeAddr, 2)
        nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment
        assert.equal("0x6", nonce)

        // send three more mixed txs without await
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.set(8888).send().then().catch(function(e) {
            assert.fail("transaction reverted: " + err);
        })
        contract.methods.err().send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 2)
        nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment
        assert.equal("0x9", nonce)
    })

 })