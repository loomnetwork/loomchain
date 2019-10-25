const Web3 = require('web3')
const fs = require('fs')
const path = require('path')
const bip39 = require('bip39')
const hdkey = require('ethereumjs-wallet/hdkey')
const {
    Client, NonceTxMiddleware, SignedTxMiddleware, Address, LocalAddress, CryptoUtils,
    Contracts
} = require('loom-js')
// TODO: fix this export in loom-js
const { OfflineWeb3Signer } = require('loom-js/dist/solidity-helpers')

const AddressMapper = Contracts.AddressMapper

async function mapAccounts({ client, signer, ethAddress, loomAddress }) {
    const ethAccountAddr = Address.fromString(`eth:${ethAddress}`)
    const loomAccountAddr = Address.fromString(`${client.chainId}:${loomAddress}`)
    const mapperContract = await AddressMapper.createAsync(client, loomAccountAddr)
    
    try {
      const mapping = await mapperContract.getMappingAsync(loomAccountAddr)
      console.log(`${mapping.from.toString()} is already mapped to ${mapping.to.toString()}`)
      return
    } catch (err) {
      // assume this means there is no mapping yet, need to fix loom-js not to throw in this case
    }
    console.log(`mapping ${ethAccountAddr.toString()} to ${loomAccountAddr.toString()}`)
    await mapperContract.addIdentityMappingAsync(loomAccountAddr, ethAccountAddr, signer)
    console.log(`Mapped ${loomAccountAddr} to ${ethAccountAddr}`)
  }

async function generateEthAccounts(numAccounts, mnemonic) {
    if (!mnemonic) {
        mnemonic = bip39.generateMnemonic()
    }

    console.log('using mnemonic: ' + mnemonic)

    const seed = await bip39.mnemonicToSeed(mnemonic)
    const hdWallet = hdkey.fromMasterSeed(seed)
    const hdPath = "m/44'/60'/0'/0/"

    const wallets = []
    for (let i = 0; i < numAccounts; i++) {
        wallets.push(hdWallet.derivePath(hdPath + i.toString()).getWallet())
    }
    fs.writeFileSync(path.join(__dirname, `../eth_mnemonic`), mnemonic)
    return wallets
}

function generateLoomAccounts(numAccounts) {
    const accounts = []
    for (let i = 0; i < numAccounts; i++) {
        const privKey = CryptoUtils.generatePrivateKey();
        const pubKey = CryptoUtils.publicKeyFromPrivateKey(privKey);
        accounts.push({
            privateKey: privKey,
            publicKey: pubKey,
            address: LocalAddress.fromPublicKey(pubKey).toString()
        })
    }
    return accounts
}

/**
 * Generates a bunch of Loom & Ethereum private keys and maps the corresponding accounts via the
 * first node of the cluster found at the location specified by the CLUSTER_DIR env var.
 * The first account has a fixed Loom private key (found in the private_key file in the parent dir).
 */
async function main() {
    if (!process.env.CLUSTER_DIR) {
        throw new Error('CLUSTER_DIR env var not defined')
    }
    const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
    const mnemonic = process.argv[2] // can be passed in as the first parameter to the script
    const numAccounts = 5

    let errored = false
    let client
    try {
        const web3js = new Web3(`http://${nodeAddr}/eth`)
        //web3js.eth.accounts.wallet.add(ethAccount)
        const loomPrivateKeyStr = fs.readFileSync(path.join(__dirname, '../private_key'), 'utf-8')
        const loomPrivateKey = CryptoUtils.B64ToUint8Array(loomPrivateKeyStr)
        const loomPublicKey = CryptoUtils.publicKeyFromPrivateKey(loomPrivateKey)
        const loomAccounts = [{
            privateKey: loomPrivateKey,
            publicKey: loomPublicKey,
            address: LocalAddress.fromPublicKey(loomPublicKey).toString()
        }]
        
        const client = new Client(
            'default',
            `ws://${nodeAddr}/websocket`,
            `ws://${nodeAddr}/queryws`
        )
        
        client.on('error', msg => {
            console.error('Loom connection error', msg)
        })
 
        loomAccounts.push(...generateLoomAccounts(numAccounts))
        const ethAccounts = await generateEthAccounts(numAccounts + 1, mnemonic)
        for (let i = 0; i < numAccounts + 1; i++) {
            client.txMiddleware = [
                new NonceTxMiddleware(loomAccounts[i].publicKey, client),
                new SignedTxMiddleware(loomAccounts[i].privateKey)
            ]
            const ethAccount = web3js.eth.accounts.privateKeyToAccount(
                '0x' + ethAccounts[i].getPrivateKey().toString('hex')
            )
            const signer = new OfflineWeb3Signer(web3js, ethAccount)
            await mapAccounts({
                client,
                signer,
                ethAddress: ethAccount.address,
                loomAddress: loomAccounts[i].address
            })
        }
    } catch (err) {
      console.error(err)
      errored = true
    } finally {
      if (client) {
        client.disconnect()
      }
    }
    process.exit(errored ? 1 : 0)
}

main()