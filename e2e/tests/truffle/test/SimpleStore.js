const { assertRevert, delay, waitForXBlocks, getNonce } = require('./helpers')
const Web3 = require('web3')
const rp = require('request-promise')
const fs = require('fs')
const path = require('path')

const SimpleStore = artifacts.require('SimpleStore')

const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js')


contract('SimpleStore', async (accounts) => {
    let web3js

    beforeEach(async () => {
        web3js = new Web3(web3.currentProvider)
    })

    it('SimpleStore has been deployed', async () => {
        const simpleStoreContract = await SimpleStore.deployed()
        assert(simpleStoreContract.address)
    })

    it('SimpleStore test set/get', async () => {
        const value = 777
        const simpleStoreContract = await SimpleStore.deployed()
        const tx = await simpleStoreContract.set(value)
        const data = await simpleStoreContract.get()
        assert.equal(data, value)
    })

    it('SimpleStore test increment nonce', async () => {
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

        const simpleStoreContract = await SimpleStore.deployed()
        const contract = new web3.eth.Contract(SimpleStore._json.abi, simpleStoreContract.address, {from});

        // send three successful txs
        try {await contract.methods.set(1111).send()} catch(err) {}
        try {await contract.methods.set(2222).send()} catch(err) {}
        try {await contract.methods.set(3333).send()} catch(err) {}
        
        await waitForXBlocks(nodeAddr, 1)
        let nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment 
        assert.equal("0x3",nonce)

        // send three more successful txs without await
        contract.methods.set(4444).send().then().catch(function(e) {})
        contract.methods.set(5555).send().then().catch(function(e) {})
        contract.methods.set(6666).send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 1)
        nonce = await getNonce(nodeAddr, from)
        // expect nonce to increment
        assert.equal("0x6",nonce)
    })

})
