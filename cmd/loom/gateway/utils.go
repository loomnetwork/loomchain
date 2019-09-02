package gateway

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/pkg/errors"
)

// Converts a bech32 encoded Binance address string to a hex address string.
func binanceAddressToHexAddress(bech32str string) (string, error) {
	if len(bech32str) == 0 {
		return "", errors.New("bech32str address can't be empty")
	}

	var prefix string
	if strings.HasPrefix(bech32str, "bnb") {
		prefix = "bnb"
	} else if strings.HasPrefix(bech32str, "tbnb") {
		prefix = "tbnb"
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
		return "", fmt.Errorf("invalid bech32 prefix (%s != %s)", prefix, hrp)
	}

	return "0x" + hex.EncodeToString(addressBytes), nil
}
