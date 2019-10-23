const Web3 = require('web3')
const fs = require('fs')
const path = require('path')
const {
    Client, NonceTxMiddleware, SignedTxMiddleware, Address, LocalAddress, CryptoUtils, LoomProvider,
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

async function main() {
    if (!process.env.CLUSTER_DIR) {
        throw new Error('CLUSTER_DIR env var not defined')
    }
    const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()

    let mapped = false
    let client
    try {
        const ethPrivateKey = fs.readFileSync(path.join(__dirname, '../eth_private_key'), 'utf-8')
        const web3js = new Web3(`http://${nodeAddr}/eth`)
        const ethAccount = web3js.eth.accounts.privateKeyToAccount('0x' + ethPrivateKey)
        web3js.eth.accounts.wallet.add(ethAccount)
        
        const loomPrivateKeyStr = fs.readFileSync(path.join(__dirname, '../private_key'), 'utf-8')
        const loomPrivateKey = CryptoUtils.B64ToUint8Array(loomPrivateKeyStr)
        const loomPublicKey = CryptoUtils.publicKeyFromPrivateKey(loomPrivateKey)
        const client = new Client(
            'default',
            `ws://${nodeAddr}/websocket`,
            `ws://${nodeAddr}/queryws`
        )
        client.txMiddleware = [
            new NonceTxMiddleware(loomPublicKey, client),
            new SignedTxMiddleware(loomPrivateKey)
        ]
        client.on('error', msg => {
            console.error('Loom connection error', msg)
        })
 
        const signer = new OfflineWeb3Signer(web3js, ethAccount)
        await mapAccounts({
            client,
            signer,
            ethAddress: ethAccount.address,
            loomAddress: LocalAddress.fromPublicKey(loomPublicKey).toString()
        })
        mapped = true
    } catch (err) {
      console.error(err)
    } finally {
      if (client) {
        client.disconnect()
      }
    }
    process.exit(mapped ? 0 : 1)
}

main()