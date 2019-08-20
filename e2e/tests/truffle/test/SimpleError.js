const { assertRevert, delay, waitForXBlocks, getNonce } = require('./helpers')
const Web3 = require('web3')
const rp = require('request-promise')
const fs = require('fs')
const path = require('path')

 const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js')

 const SimpleError = artifacts.require('SimpleError')

 contract('SimpleError', async (accounts) => {
    it('SimpleError has been deployed', async () => {
        const simpleStoreContract = await SimpleError.deployed()
        assert(simpleStoreContract.address)
    })

     it('Increment nonce for failed txs', async () => {
        const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        const chainID = 'default'
        const writeUrl = `ws://${nodeAddr}/websocket`
        const readUrl = `ws://${nodeAddr}/queryws`

        const privateKey = CryptoUtils.generatePrivateKey()
        const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)
        const from = LocalAddress.fromPublicKey(publicKey).toString()
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
        const web3 = new Web3(loomProvider)

        const simpleErrorContract = await SimpleError.deployed()
        const contract = new web3.eth.Contract(SimpleError._json.abi, simpleErrorContract.address, {from});
        // send three reverted txs
        try {await contract.methods.err().send()} catch(err) {}
        try {await contract.methods.err().send()} catch(err) {}
        try {await contract.methods.err().send()} catch(err) {}

        await waitForXBlocks(nodeAddr, 1)
        let nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment even if the txs reverted
        assert.equal("0x3",nonce)

         // send three more reverted txs without await
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.err().send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 1)
        nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment even if the txs reverted
        assert.equal("0x6",nonce)
    })

 })