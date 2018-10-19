const { assertRevert } = require('./helpers')

const MyToken = artifacts.require('MyToken')

contract('MyToken', async (accounts) => {
  let alice, bob, dan, trudy, eve

  beforeEach(async () => {
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
})
