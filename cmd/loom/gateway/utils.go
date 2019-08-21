package gateway

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/pkg/errors"
)

func bech32ToHex(bech32str string) (string, error) {
	var prefix string
	if strings.HasPrefix(bech32str, "bnb") {
		prefix = "bnb"
	} else if strings.HasPrefix(bech32str, "tbnb") {
		prefix = "tbnb"
	}

	if len(bech32str) == 0 {
		return "", errors.New("decoding bech32 address failed: must provide an address")
	}
	hrp, data, err := bech32.Decode(bech32str)
	if err != nil {
		return "", err
	}
	addressBytes, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", err
	}

	if hrp != prefix {
		return "", fmt.Errorf("invalid bech32 prefix. Expected %s, Got %s", prefix, hrp)
	}

	hexAddr := "0x" + hex.EncodeToString(addressBytes)
	return hexAddr, nil
}
