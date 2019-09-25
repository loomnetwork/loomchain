const Web3 = require('web3')
const fs = require('fs')
const path = require('path')
const EthereumTx = require('ethereumjs-tx').Transaction
const { getLoomEvmTxHash } = require('./helpers')

const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js')

const TxHashTestContract = artifacts.require('TxHashTestContract')

 contract('TxHashTestContract', async (accounts) => {
    let contract, fromAddr, nodeAddr, txHashTestContract

    beforeEach(async () => {
        nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        const chainID = 'default'
        const writeUrl = `ws://${nodeAddr}/websocket`
        const readUrl = `ws://${nodeAddr}/queryws`

        const privateKey = CryptoUtils.generatePrivateKey()
        const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)

        fromAddr = LocalAddress.fromPublicKey(publicKey)
        const from = fromAddr.toString()

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
        txHashTestContract = await TxHashTestContract.deployed()
        contract = new web3.eth.Contract(TxHashTestContract._json.abi, txHashTestContract.address, {from});
    })

     it('Test tx hashes match', async () => {
        let txParams = {
            nonce: '0x1', //expect nonce to be 1
            gasPrice: '0x0', // gas price is always 0
            gasLimit: '0xFFFFFFFFFFFFFFFF', // gas limit right now is max.Uint64
            to: txHashTestContract.address,
            value: '0x0',
            data: '0x60fe47b10000000000000000000000000000000000000000000000000000000000000457', // set(1111)
        }

        let tx = new EthereumTx(txParams)
        let expectedTxHash = getLoomEvmTxHash(tx, fromAddr)
       
        try {
            var txResult = await contract.methods.set(1111).send()
            assert.equal(txResult.transactionHash, '0x'+expectedTxHash)
        } catch(err) {
            assert.fail("transaction reverted: " + err);
        }

        txParams = {
            nonce: '0x2', //expect nonce to be 2
            gasPrice: '0x0', // gas price is always 0
            gasLimit: '0xFFFFFFFFFFFFFFFF', // gas limit right now is max.Uint64
            to: txHashTestContract.address,
            value: '0x0',
            data: '0x60fe47b100000000000000000000000000000000000000000000000000000000000008AE', // set(2222)
        }

        tx = new EthereumTx(txParams)
        expectedTxHash = getLoomEvmTxHash(tx, fromAddr)
        
        try {
            var txResult = await contract.methods.set(2222).send()
            assert.equal(txResult.transactionHash, '0x'+expectedTxHash)
        } catch(err) {
            assert.fail("transaction reverted: " + err);
        }
    })
 })