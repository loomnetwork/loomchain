// +build evm

package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/tendermint/btcd/btcec"
)

var (
	// maximum number of event per fetching
	eventSize               = 20
	ErrTronContractNotFound = errors.New("contract not found")
)

// TronClient defines typed wrappers for the Tron HTTP API.
// https://github.com/tronprotocol/tron-grid
type TronClient struct {
	url            string
	client         *http.Client
	eventPollDelay time.Duration // delay time to prevent spamming event server
}

// ConnectToTron connects a client to the given raw url.
func ConnectToTron(rawurl string, oracleEventPollDelay time.Duration) (*TronClient, error) {
	_, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return &TronClient{
		url: rawurl,
		client: &http.Client{
			Timeout: 5 * time.Second, // set default timeout to 5 seconds
		},
		eventPollDelay: oracleEventPollDelay,
	}, nil
}

func (c *TronClient) GetLastBlockNumber(ctx context.Context) (uint64, error) {
	resp, err := c.client.Post(fmt.Sprintf("%s/wallet/getnowblock", c.url), "text/json", nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// build a custom block header struct to read only block number
	var block = struct {
		BlockHeader struct {
			RawData struct {
				Number int64 `json:"number"`
			} `json:"raw_data"`
		} `json:"block_header"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&block); err != nil {
		return 0, err
	}

	return uint64(block.BlockHeader.RawData.Number), nil
}

type TronContract struct {
	ContractAddress string `json:"contract_address"`
	OriginalAddress string `json:"origin_address"`
}

func (c *TronClient) GetContract(ctx context.Context, contractAddress string) (*TronContract, error) {
	req := struct {
		Value string `json:"value"`
	}{
		Value: contractAddress,
	}
	data, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Post(fmt.Sprintf("%s/wallet/getcontract", c.url), "text/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tronContract TronContract
	if err := json.NewDecoder(resp.Body).Decode(&tronContract); err != nil {
		return nil, err
	}

	if tronContract.ContractAddress == "" || tronContract.OriginalAddress == "" {
		return nil, ErrTronContractNotFound
	}

	return &tronContract, nil
}

type tronEvent struct {
	TransactionID string            `json:"transaction_id"`
	EventIndex    uint              `json:"event_index"`
	BlockNumber   uint64            `json:"block_number"`
	Result        map[string]string `json:"result"`
	ResultType    map[string]string `json:"result_type"`
	Fingerprint   string            `json:"_fingerprint"`
}

func (c *TronClient) getEventsByBlockNumber(contractAddress, eventName string, blockNumber uint64) ([]tronEvent, error) {
	u := fmt.Sprintf("%s/event/contract/%s/%s/%d", c.url, contractAddress, eventName, blockNumber)
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []tronEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}

	return events, nil
}

func (c *TronClient) getEventsByFingerprint(contractAddress, eventName string, fingerprint string) ([]tronEvent, error) {
	u := fmt.Sprintf("%s/event/contract/%s/%s?size=%d&fingerprint=%s", c.url, contractAddress, eventName, eventSize, fingerprint)
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []tronEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}

	return events, nil
}

func (c *TronClient) filterEvents(contractAddress string, eventName string, fromBlock, toBlock uint64) ([]tronEvent, error) {
	var events []tronEvent
	var fingerprint string
	// tron uses base58 address to query event
	// TODO: maybe move this helper to go-loom
	base58Addr := AddressHexToBase58(contractAddress)
MAIN_LOOP:
	for {
		filteredEvents, err := c.getEventsByFingerprint(base58Addr, eventName, fingerprint)
		if err != nil {
			return nil, err
		}
		if len(filteredEvents) == 0 {
			break
		}
		for i, event := range filteredEvents {
			if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
				events = append(events, event)
			}
			if event.BlockNumber < fromBlock {
				break MAIN_LOOP
			}
			// grab fingerprint from the last event
			if i == len(filteredEvents)-1 {
				fingerprint = event.Fingerprint
			}
		}

		if fingerprint == "" {
			break
		}

		// add slightly delay to prevent spamming the event server
		time.Sleep(c.eventPollDelay)
	}
	return events, nil
}

func (c *TronClient) FilterTRXReceived(contractAddress string, fromBlock, toBlock uint64) ([]tronEvent, error) {
	return c.filterEvents(contractAddress, "TRXReceived", fromBlock, toBlock)
}

func (c *TronClient) FilterTRC20Received(contractAddress string, fromBlock, toBlock uint64) ([]tronEvent, error) {
	return c.filterEvents(contractAddress, "TRC20Received", fromBlock, toBlock)
}

func (c *TronClient) FilterTokenWithdrawn(contractAddress string, fromBlock, toBlock uint64) ([]tronEvent, error) {
	return c.filterEvents(contractAddress, "TokenWithdrawn", fromBlock, toBlock)
}

// Utils for Tron
// Implementation see: https://github.com/tronprotocol/tron-web/blob/04a96d826c1dc084c73689d7db6d1ddab2f26810/src/utils/crypto.js

func AddressHexToBase58(hex string) string {
	if strings.HasPrefix(hex, "0x") {
		hex = strings.TrimPrefix(hex, "0x")
		hex = fmt.Sprintf("41%s", hex)
	}
	addressBytes := common.FromHex(hex)
	hash0 := SHA256(addressBytes)
	hash1 := SHA256(hash0)
	checkSum0 := hash1[0:4]
	checkSum1 := append(addressBytes, checkSum0...)
	return base58.Encode(checkSum1)
}

func AddressBase58ToHex(b58 string) (string, error) {
	if len(b58) <= 4 {
		return "", errors.New("invalid base58 string length")
	}
	address := base58.Decode(b58)
	offset := len(address) - 4
	checksum0 := address[offset:]
	address = address[0:offset]
	hash0 := SHA256(address)
	hash1 := SHA256(hash0)
	checksum1 := hash1[0:4]
	if checksum0[0] == checksum1[0] && checksum0[1] == checksum1[1] &&
		checksum0[2] == checksum1[2] && checksum0[3] == checksum1[3] {
		d := hex.EncodeToString(address)
		if len(d) == 0 {
			return "0", nil
		}
		return d, nil
	}
	return "", errors.New("invalid address provided")
}

func AddressBase58FromPrivKey(key string) string {
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(key))
	ekey := privKey.ToECDSA()
	address := crypto.PubkeyToAddress(ekey.PublicKey).Hex()
	return AddressHexToBase58(address)
}

func SHA256(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}
