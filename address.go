package loom

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"
)

type LocalAddress [20]byte

// From ethereum with the new sha3
func (a LocalAddress) Hex() string {
	unchecksummed := hex.EncodeToString(a[:])
	sha := sha3.New256()
	sha.Write([]byte(unchecksummed))
	hash := sha.Sum(nil)

	result := []byte(unchecksummed)
	for i := 0; i < len(result); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return string(result)
}

func (a LocalAddress) String() string {
	return "0x" + a.Hex()
}

type Address struct {
	ChainID string
	Local   LocalAddress
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%s", a.ChainID, a.Local.String())
}
