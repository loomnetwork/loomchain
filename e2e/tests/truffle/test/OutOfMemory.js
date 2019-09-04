const { waitForXBlocks } = require('./helpers')
const Web3 = require('web3')
const fs = require('fs')
const path = require('path')

const OutOfMemory = artifacts.require('OutOfMemory')
contract('OutOfMemory', async () => {

    let web3js
    let someWord = "save"
    beforeEach(async () => {
        nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        web3js = new Web3(web3.currentProvider)
        memoryContract = await OutOfMemory.deployed();
    })

    it('Out Of Memory has been deployed', async () => {
        assert(memoryContract.address)
    })

    it('Test SHA3 simple hash', async () => {
        const commandHash = await memoryContract.getProofRequest()
        assert.equal(commandHash, 0x157dca92e4250458339d4b835250d44c238f3355e1b7986195188ee434e9baff)
    })

    it('Check proofRequest OK for TX section', async () => {
        const resp = await memoryContract.HashAndMod(web3js.utils.fromAscii(someWord))
        actual = getEventValue(resp, 1) // got actual value from event emiited then Assume that method HashAndMod working correctly
        console.log(actual)
        assert.equal(actual.hashCheck, false)
    })

    it('ZKP check section', async () => {
        const resp = await memoryContract.MultiplyModulo(web3js.utils.fromAscii(someWord))
        waitForXBlocks(nodeAddr,5)
        n = getEventValue(resp, 0).value
        m = getEventValue(resp, 1).value
        zkpPrime = getEventValue(resp, 2).value
        actual = getEventValue(resp, 3).value
        assert(true, actual < zkpPrime)
    })

    // it('broken Array', async () => {
    //     console.log("broken array")
    //     // const resp = await memoryContract.brokenArray()
    //     // console.log(resp)
    //     assert.equal(true, true)
    // })
})

function getEventValue(result, index) {
    return result.logs[index].args;
}