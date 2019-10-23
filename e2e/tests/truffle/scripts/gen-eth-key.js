// This script generates a new BIP39 mnemonic and writes it out to a file in the parent directory,
// it also generates the a key from the mnemonic and writes that out to a file in the parent
// directory. The script expects 1-2 arguments, the first must specify the prefix to use for the
// generated files, the second argument may be used to specify the mnemonic to use instead of
// generating a new one.

const fs = require('fs')
const path = require('path')
const bip39 = require('bip39')
const hdkey = require('ethereumjs-wallet/hdkey')

async function main() {
const prefix = process.argv[2]

if (!prefix) {
    throw new Error('prefix not specified')
}

let mnemonic = process.argv[3]

if (!mnemonic) {
    mnemonic = bip39.generateMnemonic()
}

console.log('using mnemonic: ' + mnemonic)

const seed = await bip39.mnemonicToSeed(mnemonic)
const hdwallet = hdkey.fromMasterSeed(seed)
const wallet_hdpath = "m/44'/60'/0'/0/"

const wallet = hdwallet.derivePath(wallet_hdpath + '0').getWallet()

fs.writeFileSync(path.join(__dirname, `../${prefix}_account`), '0x' + wallet.getAddress().toString('hex'))
fs.writeFileSync(path.join(__dirname, `../${prefix}_mnemonic`), mnemonic)
fs.writeFileSync(path.join(__dirname, `../${prefix}_private_key`), wallet.getPrivateKey().toString('hex'))
}

main()