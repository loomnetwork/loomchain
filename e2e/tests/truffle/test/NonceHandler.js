const { waitForXBlocks, ethGetTransactionCount } = require('./helpers')
const Web3 = require('web3')
const fs = require('fs')
const path = require('path')
const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client, EthersSigner,
    createDefaultTxMiddleware, Address, LocalAddress, CryptoUtils, LoomProvider, Contracts
} = require('loom-js')
const ethers = require('ethers').ethers

const NonceTestContract = artifacts.require('NonceTestContract')

contract.skip('NonceTestContract', async (accounts) => {
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

        const setupMiddlewareFn = function(client, privateKey) {
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
            { from: localAddr.local.toString() }
        );
    })

    it('Test nonce handler with failed txs', async () => {
        const initialNonce = await ethGetTransactionCount(nodeAddr, from)
        // send three reverted txs
        var deliveredTxCount = 0;
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            // Expect evm reverted error
            if (err.toString().includes("reverted")) {
                deliveredTxCount++
            } else {
                console.log(err)
            }
            assert(err)
        }
        await waitForXBlocks(nodeAddr, 1)
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            // Expect evm reverted error
            if (err.toString().includes("reverted")) {
                deliveredTxCount++
            } else {
                console.log(err)
            }
            assert(err)
        }
        await waitForXBlocks(nodeAddr, 1)
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            // Expect evm reverted error
            if (err.toString().includes("reverted")) {
                deliveredTxCount++
            } else {
                console.log(err)
            }
            assert(err)
        }

        await waitForXBlocks(nodeAddr, 2)
        let nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment even if the txs reverted
        assert.equal(deliveredTxCount, parseInt(nonce, 16) - parseInt(initialNonce, 16))

         // send three more reverted txs without await
        contract.methods.err().send().then().catch(function(err) {
            if (!err.toString().includes("reverted")) {
                console.log("unexpected err:", err)
            }
        })
        contract.methods.err().send().then().catch(function(err) {
            if (!err.toString().includes("reverted")) {
                console.log("unexpected err:", err)
            }
        })
        contract.methods.err().send().then().catch(function(err) {
            if (!err.toString().includes("reverted")) {
                console.log("unexpected err:", err)
            }
        })

        await waitForXBlocks(nodeAddr, 2)
        nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment even if the txs reverted
        assert.equal(6, parseInt(nonce, 16) - parseInt(initialNonce, 16))
    })

    it('Test nonce handler with successful txs', async () => {
        const initialNonce = await ethGetTransactionCount(nodeAddr, from)
        // send three successful txs
        try {
            await contract.methods.set(1111).send()
            await contract.methods.set(2222).send()
            await contract.methods.set(3333).send()
        } catch(err) {
            assert.fail("transaction reverted:", err);
        }
        
        await waitForXBlocks(nodeAddr, 2)
        let nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment 
        assert.equal(3, parseInt(nonce, 16) - parseInt(initialNonce, 16))

        // send three more successful txs without await
        contract.methods.set(4444).send().then().catch(function(e) {})
        contract.methods.set(5555).send().then().catch(function(e) {})
        contract.methods.set(6666).send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 2)
        nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment
        assert.equal(6, parseInt(nonce, 16) - parseInt(initialNonce, 16))
    })

    it('Test nonce handler with mixed txs', async () => {
        const initialNonce = await ethGetTransactionCount(nodeAddr, from)
        // send a mix of successful & failed txs
        try {
            await contract.methods.set(1111).send()
        } catch(err) {
            assert.fail("transaction reverted: " + err);
        }
        await waitForXBlocks(nodeAddr, 1)
        try {
            await contract.methods.err().send()
            assert.fail("transaction is supposed to revert")
        } catch(err) {
            if (!err.toString().includes("reverted")) {
                assert.fail("transaction reverted with unexpected error: " + err);
            }
            assert(err)
        }
        await waitForXBlocks(nodeAddr, 1)
        try {
            await contract.methods.set(2222).send()
        } catch(err) {
            assert.fail("transaction reverted: " + err);
        }
        
        await waitForXBlocks(nodeAddr, 2)
        let nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment 
        assert.equal(3, parseInt(nonce, 16) - parseInt(initialNonce, 16))

        // send three more mixed txs without await
        contract.methods.set(4444).send().then().catch(function(e) {
            assert.fail("transaction reverted: " + err);
        })
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.set(6666).send().then().catch(function(e) {
            assert.fail("transaction reverted: " + err);
        })

        await waitForXBlocks(nodeAddr, 2)
        nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment
        assert.equal(6, parseInt(nonce, 16) - parseInt(initialNonce, 16))

        // send three more mixed txs without await
        contract.methods.err().send().then().catch(function(e) {})
        contract.methods.set(8888).send().then().catch(function(e) {
            assert.fail("transaction reverted: " + err);
        })
        contract.methods.err().send().then().catch(function(e) {})

        await waitForXBlocks(nodeAddr, 2)
        nonce = await ethGetTransactionCount(nodeAddr, from)
        // expect nonce to increment
        assert.equal(9, parseInt(nonce, 16) - parseInt(initialNonce, 16))
    })

})