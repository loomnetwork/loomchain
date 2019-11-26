
const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const {
  createDefaultTxMiddleware, Client, Address, LocalAddress, CryptoUtils, Contracts, EthersSigner
} = require('loom-js')
const ethers = require('ethers').ethers
const { getContractFuncInterface, getLatestBlock, getMappedAccount } = require('./helpers')
const MyToken = artifacts.require('MyToken');
const { waitForXBlocks } = require('./helpers');

// web3 functions called using truffle objects use the loomProvider
// web3 functions called uisng we3js access the loom QueryInterface directly
contract('MyToken', async (accounts) => {
  let web3js, nodeAddr, alice, aliceLoomAddr, bob, bobLoomAddr

  beforeEach(async () => {
    if (!process.env.CLUSTER_DIR) {
      throw new Error('CLUSTER_DIR env var not defined');
    }
    nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8');
    web3js = new Web3(new Web3.providers.HttpProvider(`http://${nodeAddr}/eth`));
    const loomPrivateKeyStr = fs.readFileSync(path.join(__dirname, '..', 'private_key'), 'utf-8')
    const loomPrivateKey = CryptoUtils.B64ToUint8Array(loomPrivateKeyStr)
    const loomPublicKey = CryptoUtils.publicKeyFromPrivateKey(loomPrivateKey)
    const loomCallerAddr = LocalAddress.fromPublicKey(loomPublicKey).toString()

    alice = accounts[1];
    bob = accounts[2];

    // When using the Truffle HDWallet provider the accounts have Ethereum addresses
    if (process.env.TRUFFLE_PROVIDER === 'hdwallet') {
      aliceLoomAddr = await getMappedAccount(nodeAddr, loomCallerAddr, alice)
      bobLoomAddr = await getMappedAccount(nodeAddr, loomCallerAddr, bob)
    } else { // when using Loom Truffle provider the accounts have DAppChain addresses
      aliceLoomAddr = alice
      bobLoomAddr = bob
    }
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

      const curBlock = await getLatestBlock(nodeAddr)
      const maxBlock = 20

      const myTokenLogs = await web3js.eth.getPastLogs({
        address: tokenContract.address,
        fromBlock: curBlock - maxBlock,
        toBlock: curBlock,
      });
      for (let i=0 ; i<myTokenLogs.length ; i++) {
        assert.equal(myTokenLogs[i].address.toLowerCase(), tokenContract.address.toLowerCase(), "log address and contract address")
      }

      const aliceLogs = await web3js.eth.getPastLogs({
        topics: [null, null, web3js.utils.padLeft(alice, 64), null],
        fromBlock: curBlock - maxBlock,
        toBlock: curBlock,
      });
      for (let i=0 ; i<aliceLogs.length ; i++) {
        assert.equal(aliceLogs[i].topics[2], web3js.utils.padLeft(alice, 64), "log address topic and caller")
      }
  });

  it('eth_getTransactionReceipt', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(101, { from: alice });
    if (process.env.TRUFFLE_PROVIDER === 'hdwallet') {
      // NOTE: This is a bug in EvmTxReceipt (/query) that's been fixed in EthGetTransactionReceipt (/eth)
      assert.equal(result.receipt.contractAddress, null, "contract address on receipt should be null");
    }

    const receipt = await web3js.eth.getTransactionReceipt(result.tx);
    assert.equal(receipt.contractAddress, null, "contract address from deploy tx and receipt");
    assert.equal(receipt.to.toLowerCase(), tokenContract.address.toLowerCase(), "receipt to matches contract address from deploy tx");
    assert.equal(receipt.from.toLowerCase(), aliceLoomAddr.toLowerCase(), "receipt from and caller");
    assert.equal(receipt.logs.length, 1, "number of logs");
    assert.equal(receipt.logs[0].topics.length, 4, "number of topics in log");
    assert.equal(receipt.logs[0].address.toLowerCase(), tokenContract.address.toLowerCase(), "log address");
  });

  it('eth_getTransactionByHash', async () => {
    const tokenContract = await MyToken.deployed();
    const result = await tokenContract.mintToken(102, { from: alice });
    const txObj = await web3js.eth.getTransaction(result.tx);
    await waitForXBlocks(nodeAddr, 1)
    assert.equal(txObj.to.toLowerCase(), tokenContract.address.toLowerCase(), "transaction object to address and contract address");
    //console.log("piers txObj.transactions[0].from",txObj.transactions[0].from)
    console.log("piers result",result)
    console.log("piers result.receipt",result.receipt)
    console.log("piers result.receipt.from",result.receipt.from)
    assert.equal(txObj.from.toLowerCase(), result.receipt.from.toLowerCase(), "transaction object from address and caller");
    assert.equal(result.receipt.from.toLowerCase(), result.receipt.from.toLowerCase(), "receipt from and caller");
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
    await waitForXBlocks(nodeAddr, 1)
    const txObject = await web3js.eth.getTransaction(result.tx, true);
    const blockByHash = await web3js.eth.getBlock(txObject.blockHash, true);

    assert.equal(txObject.blockHash, blockByHash.hash, "tx object hash and block hash");
    assert.equal(txObject.blockNumber, blockByHash.number, "receipt block number and block object number");

    assert.equal(blockByHash.transactions.length, 1, "block transaction count");
    console.log("piers blockByHash.transactions[0].from",blockByHash.transactions[0].from)
    console.log("piers result",result)
    console.log("piers result.receipt",result.receipt)
    console.log("piers result.receipt.from",result.receipt.from)
    assert.equal(blockByHash.transactions[0].from.toLowerCase(), result.receipt.from.toLowerCase(), "caller and block transaction from");
    assert.equal(blockByHash.transactions[0].to.toLowerCase(), tokenContract.address.toLowerCase(), "token address and block transaction to");
    assert.equal(txObject.blockNumber, blockByHash.transactions[0].blockNumber, "receipt block number and block transaction block bumber");
    assert.equal(txObject.hash.toLowerCase(), blockByHash.transactions[0].hash.toLowerCase(), "receipt tx hash and block transaction hash");
    assert.equal(txObject.blockHash.toLowerCase(), blockByHash.transactions[0].blockHash.toLowerCase(), "tx object block hash and block transaction block hash");

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
    console.log("result bad immediate",result)
    await waitForXBlocks(nodeAddr, 1)
    // Do second transaction to move to next block
    await tokenContract.mintToken(108, { from: alice });
    const tx1 = await web3js.eth.getTransaction(result.tx, true);
    const tx2 = await web3js.eth.getTransactionFromBlock(tx1.blockHash, 0);
    console.log("piers tx2.from",tx2.from.toLowerCase())
    console.log("piers result bad",result)
    console.log("piers result.receipt.from",result.receipt.from)
    assert.equal(tx2.from.toLowerCase(), result.receipt.from.toLowerCase(), "caller and transaction object from");
    assert.equal(tx2.to.toLowerCase(), tokenContract.address.toLowerCase(), "contract address and transaction object to");
    assert.equal(tx1.blockNumber, tx2.blockNumber, "receipt block number and transaction object block number");
    assert.equal(tx1.hash.toLowerCase(), tx2.hash.toLowerCase(), "transaction hash and transaction object hash");
    assert.equal(tx1.blockHash.toLowerCase(), tx2.blockHash.toLowerCase(), "transaction hash using getTransaction and getTransactionFromBlock");
  });

  it('eth_call', async () => {
    const tokenContract = await MyToken.deployed();
    await tokenContract.mintToken(112, { from: alice });

    let owner = await tokenContract.ownerOf.call(112);
    const ethOwner = await web3js.eth.call({
      to: tokenContract.address,
      data: "0x6352211e0000000000000000000000000000000000000000000000000000000000000070" // abi for ownerOf(12)
    },"latest");
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
    // Create a mapping between a new DAppChain account & Ethereum account, this is necessary in
    // order to match the signer address that will be recovered from the Ethereum tx to a DAppChain
    // account, without this mapping the Ethereum tx will be rejected.
    const loomAddr = new Address(client.chainId, LocalAddress.fromPublicKey(pubKey));
    const addressMapper = await Contracts.AddressMapper.createAsync(client, loomAddr);
    const ethAccount = web3js.eth.accounts.create();
    const ethWallet = new ethers.Wallet(ethAccount.privateKey);
    const ethAddr = await ethWallet.getAddress();
    await addressMapper.addIdentityMappingAsync(
      loomAddr,
      new Address('eth', LocalAddress.fromHexString(ethAddr)),
      new EthersSigner(ethWallet)
    );
    client.disconnect();

    // Encode & send the raw Eth tx
    const tokenContract = await MyToken.deployed();
    const mintTokenInterface = getContractFuncInterface(
      new web3js.eth.Contract(MyToken._json.abi, tokenContract.address), 'mintToken'
    )
    const txCount = await web3js.eth.getTransactionCount(ethAddr)
    const txParams = {
      nonce: ethers.utils.hexlify(txCount),
      gasPrice: '0x0', // gas price is always 0
      gasLimit: '0xFFFFFFFFFFFFFFFF', // gas limit right now is max.Uint64
      to: tokenContract.address,
      value: '0x0',
      data: web3js.eth.abi.encodeFunctionCall(mintTokenInterface, ['150']) // mintToken(150)
    }
    
    const payload = await web3js.eth.accounts.signTransaction(txParams, ethAccount.privateKey);
    result = await web3js.eth.sendSignedTransaction(payload.rawTransaction);
    assert.equal(result.status, true, 'tx submitted successfully');
  });
});
