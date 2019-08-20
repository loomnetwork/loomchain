package gateway

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/pkg/errors"
)

type AccAddress []byte

const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var gen = []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

func bech32ToHex(bech32Addr string) (string, error) {
	addressBytes, err := accAddressFromBech32(bech32Addr)
	if err != nil {
		return "", err
	}
	hexAddr := "0x" + hex.EncodeToString(addressBytes)
	return hexAddr, nil
}

// AccAddressFromBech32 to create an AccAddress from a bech32 string
func accAddressFromBech32(address string) (addr AccAddress, err error) {
	var prefix string
	if strings.HasPrefix(address, "bnb") {
		prefix = "bnb"
	} else if strings.HasPrefix(address, "tbnb") {
		prefix = "tbnb"
	}

	bz, err := getFromBech32(address, prefix)
	if err != nil {
		return nil, err
	}
	return AccAddress(bz), nil
}

// GetFromBech32 to decode a bytestring from a bech32-encoded string
func getFromBech32(bech32str, prefix string) ([]byte, error) {
	if len(bech32str) == 0 {
		return nil, errors.New("decoding bech32 address failed: must provide an address")
	}
	hrp, bz, err := decodeAndConvert(bech32str)
	if err != nil {
		return nil, err
	}

	if hrp != prefix {
		return nil, fmt.Errorf("invalid bech32 prefix. Expected %s, Got %s", prefix, hrp)
	}

	return bz, nil
}

//DecodeAndConvert decodes a bech32 encoded string and converts to base64 encoded bytes
func decodeAndConvert(bech string) (string, []byte, error) {
	hrp, data, err := bech32.Decode(bech)
	if err != nil {
		return "", nil, errors.Wrap(err, "decoding bech32 failed")
	}
	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", nil, errors.Wrap(err, "decoding bech32 failed")
	}
	return hrp, converted, nil
}
