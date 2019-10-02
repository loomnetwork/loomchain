const Web3 = require('web3')
const fs = require('fs')
const path = require('path')

const {
    SpeculativeNonceTxMiddleware,
    SignedTxMiddleware,
    Client,
    LocalAddress,
    CryptoUtils,
    LoomProvider
} = require('loom-js')

const EventTestContract = artifacts.require('EventTestContract')

contract('EventTestContract', async (accounts) => {
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
        const setupMiddlewareFn = function (client, privateKey) {
            const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)
            return [new SpeculativeNonceTxMiddleware(publicKey, client), new SignedTxMiddleware(privateKey)]
        }
        var loomProvider = new LoomProvider(client, privateKey, setupMiddlewareFn)

        let web3 = new Web3(loomProvider)
        let eventTestContract = await EventTestContract.deployed()
        contract = new web3.eth.Contract(EventTestContract._json.abi, eventTestContract.address, {
            from
        });
    })

    it('Test emitted events', async () => {
        try {
            var eventCount = 0
            contract.events.allEvents()
                .on('data', (event) => {
                    eventCount++
                })
            var tx = await contract.methods.set(1).send()
            assert.equal(2, tx.events.NewValueSet.length)
            assert.equal(true, (tx.events.AnotherValueSet != undefined))
            assert.equal(3, eventCount)
        } catch (err) {
            assert.fail(err)
        }
    })

})