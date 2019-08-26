const Web3 = require('web3')

const OutOfMemory = artifacts.require('OutOfMemory')
contract('OutOfMemory',async (accounts)=>{

    let web3js

    beforeEach(async ()=> {
        web3js = new Web3(web3.currentProvider)
    })

    it('Out Of Memory has been deployed',async()=>{
        const memoryContract = await OutOfMemory.deployed()
        assert(memoryContract.address)
    })

    it('Proof Request Check', async()=> {
        const memoryContract = await OutOfMemory.deployed();
        const commandHash = await memoryContract.getProofRequest()
       
        console.log(commandHash)
        assert.equal(commandHash, 0x157dca92e4250458339d4b835250d44c238f3355e1b7986195188ee434e9baff)
        const respHello = await memoryContract.renderHelloWorld()
        console.log(respHello)
        assert.equal(respHello,"helloWorld")
    })

    it('store Transaction', async()=>{
        const memoryContract = await OutOfMemory.deployed();
        console.log("before store")
        const respbool = await memoryContract.storeTransactionP1(Web3.utils.fromAscii("save"))
        console.log(respbool)
        assert.equal(respbool, true)
        const respHello = await memoryContract.renderHelloWorld()
        console.log(respHello+"Store")
        assert.equal(respHello, "helloWorld")
    })
})