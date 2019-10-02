
const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const Mycoin = artifacts.require('MyCoin');
const {
    SpeculativeNonceTxMiddleware, SignedTxMiddleware, Client,
    LocalAddress, CryptoUtils, LoomProvider
} = require('loom-js')
const { waitForXBlocks } = require('./helpers')
// web3 functions called using truffle objects use the loomProvider
// web3 functions called uisng we3js access the loom QueryInterface directly
contract('MyCoin', async (accounts) => {
    let contract, fromAddr, nodeAddr, coinContract, alice, web3js

    beforeEach(async () => {
        nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        const chainID = 'default'
        const writeUrl = `ws://${nodeAddr}/websocket`
        const readUrl = `ws://${nodeAddr}/queryws`

        const privateKey = CryptoUtils.generatePrivateKey()
        const publicKey = CryptoUtils.publicKeyFromPrivateKey(privateKey)

        fromAddr = LocalAddress.fromPublicKey(publicKey)
        alice = fromAddr.toString()

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

        bob = accounts[2];
        let web3 = new Web3(loomProvider)
        web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));
        coinContract = await Mycoin.deployed();
        contract = new web3js.eth.Contract(Mycoin._json.abi, coinContract.address, {alice});
        
    })


  it('eth_estimateGas', async () => {
    
    console.log(web3js.version)
    await coinContract.mint(alice,100000)

    let aliceBal = await contract.methods.balanceOf(alice).call({from:bob});
    console.log("ALICE ",aliceBal)
    assert.equal(aliceBal,100000,"Alice balance not correct");

    await coinContract.transfer(bob,50000,{from:alice})

    await waitForXBlocks(nodeAddr, 2)
    let bobBal = await contract.methods.balanceOf(bob).call({from:alice});
    aliceBal = await contract.methods.balanceOf(alice).call({from:bob});
    console.log("contract ADDRESS",coinContract.address)
    console.log("alice ADDRESS",alice)
    assert.equal(bobBal,50000,"bob balance not correct");

    assert.equal(aliceBal,50000,"alice balance not correct");
    
    const estGas = await contract.methods.transfer(bob,5000).estimateGas({from:alice,value:0});
    assert.equal(estGas,1000,"estimateGas checking")

   });

//   it('eth_estimateGas', async () => {
//     const coinContract = await MyToken.deployed(); 
    
//     await coinContract.mintToken(113, { from: alice });

//     const gas = await web3js.eth.estimateGas({
//       to: coinContract.address,
//       data: "0x6352211e0000000000000000000000000000000000000000000000000000000000000070", // abi for ownerOf(12)
//       value: 0
//     },"latest");
//     assert.equal(gas,722, "pass transaction gas estimate");

//     let owner = await coinContract.ownerOf.call(113);
//     const bobBalBefore = await web3js.eth.getBalance(bob)
//     const aliceBalBefore = await web3js.eth.getBalance(alice)
//     console.log("alice address", owner)
//     const bobGas = await contract.methods.transferToken(bob,113).estimateGas({value:1});
//     console.log("transfer gas",bobGas)
//     const aliceBalAfter = await web3js.eth.getBalance(alice)
//     const bobBalAfter = await web3js.eth.getBalance(bob)
//     let newowner = await coinContract.ownerOf.call(113);
//     assert.equal(owner,newowner, "Since it just estimate gas of transaction new owner must be the same");
//     assert.equal(aliceBalAfter,aliceBalBefore, "Since it just estimate gas of transaction balance must remains the same");
//     assert.equal(bobBalAfter,bobBalBefore, "Since it just estimate gas of transaction balance must remains the same");
//     console.log("current balance",aliceBalAfter)
//     console.log("current balance",bobBalAfter) 
//   });

});

