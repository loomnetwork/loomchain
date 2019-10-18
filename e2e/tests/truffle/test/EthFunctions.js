
const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const {
  createDefaultTxMiddleware, Client, Address, LocalAddress, CryptoUtils, Contracts, EthersSigner
} = require('loom-js')
const ethers = require('ethers').ethers
const { getContractFuncInterface } = require('./helpers')

const MyToken = artifacts.require('MyToken');

// web3 functions called using truffle objects use the loomProvider
// web3 functions called uisng we3js access the loom QueryInterface directly
contract('MyToken', async (accounts) => {
  let web3js, nodeAddr;

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined');
    }
    nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8');
    web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));

    alice = accounts[1];
    bob = accounts[2];
  });

  it('eth_blockNumber', async () => {
      const blockNumber = await web3js.eth.getBlockNumber();
      assert(0 < blockNumber);
  });

  it('getPastLogs', async () => {
      const tokenContract = await MyToken.deployed();
      await tokenContract.mintToken(97, { from: alice });
      await tokenContract.mintToken(98, { from: alice });
      await tokenContract.mintToken(99, { from: bob });

      const myTokenLogs = await web3js.eth.getPastLogs({
          address: tokenContract.address
      });
      for (let i=0 ; i<myTokenLogs.length ; i++) {
          assert.equal(myTokenLogs[i].address.toLowerCase(), tokenContract.address.toLowerCase(), "log address and contract address")
      }

      const aliceLogs = await web3js.eth.getPastLogs({
          topics: [null, null, web3js.utils.padLeft(alice, 64), null]
      });
      for (let i=0 ; i<aliceLogs.length ; i++) {
          assert.equal(aliceLogs[i].topics[2], web3js.utils.padLeft(alice, 64), "log address topic and caller")
      }
  });

  it('eth_getTransactionReceipt', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(101, { from: alice });
    assert.equal(tokenContract.address, result.receipt.contractAddress, "contract address and receipt contract address");

    const receipt = await web3js.eth.getTransactionReceipt(result.tx);
    assert.equal(null, receipt.contractAddress, "contract address from deploy tx and receipt");
    assert.equal(tokenContract.address.toLowerCase(), receipt.to.toLowerCase(), "contract address from deploy tx and receipt");
    assert.equal(receipt.from.toLowerCase(), alice.toLowerCase(),  "receipt to and caller");
    assert.equal(1, receipt.logs.length, "number of logs");
    assert.equal(4, receipt.logs[0].topics.length, "number of topics in log");
    assert.equal(tokenContract.address.toLowerCase(), receipt.logs[0].address.toLowerCase(), "log address");
    assert.equal(true, receipt.logs[0].blockTime.length >3)
  });

  it('eth_getTransactionByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(102, { from: alice });
    const txObj = await web3js.eth.getTransaction(result.tx);

    assert.equal(txObj.to.toLowerCase(), result.receipt.contractAddress.toLowerCase(), "transaction object to address and receipt contract address");
    assert.equal(txObj.from.toLowerCase(), alice.toLowerCase(), "transaction object from address and caller");
  });

  it('eth_getCode', async () => {
    const tokenContract = await MyToken.deployed();
    const code = await web3js.eth.getCode(tokenContract.address);
    assert.equal(tokenContract.constructor._json.deployedBytecode, code, "contract deployed bytecode and eth_getCode result")
  });

  it('eth_getBlockByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(103, { from: alice });
    await tokenContract.mintToken(104, { from: alice });

    const txObject = await web3js.eth.getTransaction(result.tx, true);

    const blockByHash = await web3js.eth.getBlock(txObject.blockHash, true);
    assert.equal(txObject.blockHash, blockByHash.hash, "tx object hash and block hash");
    assert.equal(txObject.blockNumber, blockByHash.number, "receipt block number and block object number");

    assert.equal(1, blockByHash.transactions.length, "block transaction count");
    assert.equal(alice.toLowerCase() , blockByHash.transactions[0].from.toLowerCase(), "caller and block transaction from");
    assert.equal(tokenContract.address.toLowerCase() ,blockByHash.transactions[0].to.toLowerCase(), "token address and block transaction to");
    assert.equal(txObject.blockNumber ,blockByHash.transactions[0].blockNumber, "receipt block number and block transaction block bumber");
    assert.equal(txObject.hash.toLowerCase() ,blockByHash.transactions[0].hash.toLowerCase(), "receipt tx hash and block transaction hash");
    assert.equal(txObject.blockHash.toLowerCase() ,blockByHash.transactions[0].blockHash.toLowerCase(), "tx object block hash and block transaction block hash");

    const blockByHashFalse = await web3js.eth.getBlock(txObject.blockHash, false);
    const receipt = await web3js.eth.getTransactionReceipt(blockByHashFalse.transactions[0]);
    assert.equal(txObject.hash.toLowerCase(), receipt.transactionHash.toLowerCase(), "receipt tx hash and block transaction hash, full = false");
  });

  it('eth_getBlockByNumber', async () => {
    await MyToken.deployed();
    const blockInfo = await web3js.eth.getBlock("latest");
    assert(blockInfo.number > 0, "block number must be greater than zero");
  });

  it('eth_getBlockTransactionCountByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(105, { from: alice });
    // Do second transaction to move to next block
    await tokenContract.mintToken(106, { from: alice });
    const txObject = await web3js.eth.getTransaction(result.tx, true);

    const txCount = await web3js.eth.getBlockTransactionCount(txObject.blockHash);
    assert.equal(txCount, 1, "confirm one transaction in block");
  });

  it('eth_getTransactionByBlockHashAndIndex', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(107, { from: alice });
    // Do second transaction to move to next block
    await tokenContract.mintToken(108, { from: alice });
    const tx1 = await web3js.eth.getTransaction(result.tx, true);

    const tx2 = await web3js.eth.getTransactionFromBlock(tx1.blockHash, 0);
    assert.equal(alice.toLowerCase() , tx2.from.toLowerCase(), "caller and transaction object from");
    assert.equal(tokenContract.address.toLowerCase() ,tx2.to.toLowerCase(), "contract address and transaction object to");
    assert.equal(tx1.blockNumber ,tx2.blockNumber, "receipt block number and transaction object block number");
    assert.equal(tx1.hash.toLowerCase() ,tx2.hash.toLowerCase(), "transaction hash and transaction object hash");
    assert.equal(tx1.blockHash.toLowerCase(), tx2.blockHash.toLowerCase(), "transaction hash using getTransaction and getTransactionFromBlock");
  });

  it('eth_Call', async () => {
    const tokenContract = await MyToken.deployed();
    await tokenContract.mintToken(112, { from: alice });

    let owner = await tokenContract.ownerOf.call(112);
    console.log("piers owner", owner)
    const ethOwner = await web3js.eth.call({
      to: tokenContract.address,
      data: "0x6352211e0000000000000000000000000000000000000000000000000000000000000070" // abi for ownerOf(12)
    },"latest");
    console.log("piers ethOwner", ethOwner)
    assert.equal(ethOwner.toLowerCase(), web3js.utils.padLeft(owner, 64).toLowerCase(), "result using tokenContract and eth.call");
  });

  it('eth_sendRawTransaction', async () => {
    // Map Alice's Eth account to a DAppChain account
    const client = new Client('default', `ws://${nodeAddr}/websocket`, `ws://${nodeAddr}/queryws`);
    client.on('error', msg => {
        console.error('Error on connect to client', msg);
        console.warn('Please verify if loom cluster is running');
    });
    const privKey = CryptoUtils.generatePrivateKey();
    const pubKey = CryptoUtils.publicKeyFromPrivateKey(privKey);
    client.txMiddleware = createDefaultTxMiddleware(client, privKey);
    // Create a mapping between Alice's DAppChain account & Ethereum account, this is necessary in
    // order to match the signer address that will be recovered from the Ethereum tx to a DAppChain
    // account, without this mapping the Ethereum tx will be rejected.
    const aliceLoomAddr = new Address(client.chainId, LocalAddress.fromPublicKey(pubKey));
    const addressMapper = await Contracts.AddressMapper.createAsync(client, aliceLoomAddr);
    const ethPrivateKey = '0xe331b6d69882b4cb4ea581d88e0b604039a3de5967688d3dcffdd2270c0fd109';
    const aliceEthWallet = new ethers.Wallet(ethPrivateKey);
    const aliceEthAddr = await aliceEthWallet.getAddress();
    await addressMapper.addIdentityMappingAsync(
      aliceLoomAddr,
      new Address('eth', LocalAddress.fromHexString(aliceEthAddr)),
      new EthersSigner(aliceEthWallet)
    );
    client.disconnect();

    // Encode & send the raw Eth tx
    const tokenContract = await MyToken.deployed();
    const mintTokenInterface = getContractFuncInterface(
      new web3js.eth.Contract(MyToken._json.abi, tokenContract.address), 'mintToken'
    )
    const txParams = {
      nonce: '0x2', // identity mapping tx used nonce 1, so this tx must have nonce 2
      gasPrice: '0x0', // gas price is always 0
      gasLimit: '0xFFFFFFFFFFFFFFFF', // gas limit right now is max.Uint64
      to: tokenContract.address,
      value: '0x0',
      data: web3js.eth.abi.encodeFunctionCall(mintTokenInterface, ['150']) // mintToken(150)
    }
    
    const payload = await web3js.eth.accounts.signTransaction(txParams, ethPrivateKey);
    result = await web3js.eth.sendSignedTransaction(payload.rawTransaction);
    assert.equal(result.status, true, 'tx submitted successfully');
  });
});
