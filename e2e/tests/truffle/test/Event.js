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
    let contract, from, nodeAddr, web3js, contractAddress

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
        contractAddress = eventTestContract.address
        contract = new web3.eth.Contract(EventTestContract._json.abi, contractAddress, {
            from
        });
        web3eth = new Web3(new Web3.providers.WebsocketProvider(`ws://${nodeAddr}/eth`));
    })

    it('Test emitted events', async () => {
        try {
            var eventCount = 0
            // Test loom
            contract.events.allEvents()
                .on('data', (event) => {
                    eventCount++
                })
            // Test /eth endpoint
            var ethEventCount = 0
            web3eth.eth.subscribe('logs', {
                address: contractAddress,
            }, function (error, result) {
                if (!error) {
                    ethEventCount++
                }
            });
            var tx = await contract.methods.set(1).send()
            assert.equal(2, tx.events.NewValueSet.length)
            assert.equal(true, (tx.events.AnotherValueSet != undefined))
            assert.equal(3, eventCount)
            assert.equal(3, ethEventCount)
        } catch (err) {
            assert.fail(err)
        }

        try {
            var eventCount2 = 0
            contract.events.allEvents()
                .on('data', (event) => {
                    eventCount2++
                })
            // Test /eth endpoint
            var ethEventCount2 = 0
            web3eth.eth.subscribe('logs', {
                address: contractAddress,
            }, function (error, result) {
                if (!error) {
                    ethEventCount2++
                }
            });
            var tx = await contract.methods.set(1).send()
            assert.equal(2, tx.events.NewValueSet.length)
            assert.equal(true, (tx.events.AnotherValueSet != undefined))
            var tx = await contract.methods.set(2).send()
            assert.equal(2, tx.events.NewValueSet.length)
            assert.equal(true, (tx.events.AnotherValueSet != undefined))
            // total 6 events
            assert.equal(6, eventCount2)
            assert.equal(6, ethEventCount2)
        } catch (err) {
            assert.fail(err)
        }

        try {
            var eventCount3 = 0
            contract.events.allEvents()
                .on('data', (event) => {
                    eventCount3++
                })
            var ethEventCount3 = 0
            // Test /eth endpoint
            web3eth.eth.subscribe('logs', {
                address: contractAddress,
            }, function (error, result) {
                if (!error) {
                    ethEventCount3++
                }
            });
            // sending two txs in one block
            contract.methods.set(1).send().then()
            var tx = await contract.methods.set(2).send()
            assert.equal(2, tx.events.NewValueSet.length)
            assert.equal(true, (tx.events.AnotherValueSet != undefined))
            // total 6 events
            assert.equal(6, eventCount3)
            assert.equal(6, ethEventCount3)
        } catch (err) {
            assert.fail(err)
        }
    })

})