const { assertRevert } = require('./helpers')
const Web3 = require('web3')

const MyToken = artifacts.require('MyToken')

contract('MyToken', async (accounts) => {
    // This test doesn't really need to run with Truffle HDWallet provider, it doesn't do anything
    // all that different from EthFunctions.js ... maybe we should eliminate this test entirely.
    if (process.env.TRUFFLE_PROVIDER === 'hdwallet') {
        return
    }

    let web3js
    let alice, bob, dan, trudy, eve

    beforeEach(async () => {
        web3js = new Web3(web3.currentProvider)
        alice = accounts[1]
        bob = accounts[2]
        dan = accounts[3]
        trudy = accounts[4]
        eve = accounts[5]
    })

    it('has been deployed', async () => {
        const tokenContract = await MyToken.deployed()
        assert(tokenContract.address)
    })

    it('safeTransferFrom', async () => {
        const tokenContract = await MyToken.deployed()
        const tokens = [
            { id: 1, owner: alice },
            { id: 2, owner: alice },
            { id: 3, owner: alice },
            { id: 4, owner: alice },
            { id: 5, owner: bob },
            { id: 6, owner: bob },
            { id: 7, owner: bob },
            { id: 8, owner: bob }
        ]
        for (let i = 0; i < tokens.length; i++) {
            await tokenContract.mintToken(tokens[i].id, { from: tokens[i].owner })
            const owner = await tokenContract.ownerOf.call(tokens[i].id)
            assert.equal(owner, tokens[i].owner)
        }

        await tokenContract.transferToken(dan, 1, { from: alice })
        let owner = await tokenContract.ownerOf.call(1)
        assert.equal(owner, dan)
        await assertRevert(tokenContract.transferToken(trudy, 1, { from: alice }))
        await tokenContract.transferToken(trudy, 5, { from: bob })
        owner = await tokenContract.ownerOf.call(5)
        assert.equal(owner, trudy)
    })

    it.skip('returned receipts correctly', async () => {
        const DbSize = 10; // Should match ReceiptsLevelDbSize setting in loom.yaml
        const tokenStart = 10; // Skip over token ids used in earler tests
        const excessTokens = 5; // Extra transactions to run to ensure receipt db overflows
        const tokenContract = await MyToken.deployed();
        let txHashList = [];
        // Perform enough transactions so that receipts need to be removed from the receipt database.
        for (let tokenId = tokenStart ; tokenId < DbSize + excessTokens + tokenStart ; tokenId++ ) {
            const results = await tokenContract.mintToken(tokenId, { from: alice });
            txHashList.push(results.tx);
        }
        // Try to get receipt for transactions above
        for (let i = 0 ; i < txHashList.length ; i++ ) {
            try {
                const receipt = await web3js.eth.getTransactionReceipt(txHashList[i]);
                assert(i >= txHashList.length - DbSize); // Receipt stored in database
                assert.equal(txHashList[i], receipt.transactionHash); // tx hash matches
            }
            catch(error){
                assert(i < txHashList.length - DbSize) // Old receipt removed from database
            }
        }
    })
})
