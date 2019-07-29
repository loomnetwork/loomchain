// +build evm

package subs

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/phonkee/go-pubsub"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/rpc/lib/types"
)

const (
	allFilter = "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"address\":\"\",\"topics\":[]}"
)

var (
	subId        string
	message      []byte
	currentIndex int
	currentTopic string
	topics       = []string{
		"",
		"contract:0x8888F1F195AfA192cFee860698584C030f4c9Db1",
		"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b",
		"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b",
	}
	messageShouldBeSent = []bool{true, true, true, true}
	messageSent         = false
)

func TestUnSubscribe(t *testing.T) {
	ethSubSet := NewLegacyEthSubscriptionSet()
	conn := mockConnection{
		caller:    "myCaller",
		connected: true,
	}
	var err error
	var sub pubsub.Subscriber
	sub, subId = ethSubSet.For(conn.caller)
	sub.Do(testEthWriter(t, &conn, subId, ethSubSet))
	require.NoError(t, ethSubSet.AddSubscription(subId, "logs", allFilter))

	currentIndex = 1
	currentTopic = topics[currentIndex]
	eventData := ptypes.EventData{
		Topics: []string{strconv.Itoa(currentIndex)},
	}
	message, err = proto.Marshal(&eventData)
	require.NoError(t, err)
	ethSubSet.Publish(pubsub.NewMessage(string(message), message))
	require.True(t, messageSent)
	messageSent = false

	require.NoError(t, ethSubSet.Remove(subId))
	ethSubSet.Publish(pubsub.NewMessage(string(message), message))
	require.False(t, messageSent)
	messageSent = false
}

func TestSubscribe(t *testing.T) {
	ethSubSet := NewLegacyEthSubscriptionSet()
	conn := mockConnection{
		caller:    "myCaller",
		connected: true,
	}
	var err error
	var sub pubsub.Subscriber
	sub, subId = ethSubSet.For(conn.caller)
	sub.Do(testEthWriter(t, &conn, subId, ethSubSet))
	require.NoError(t, ethSubSet.AddSubscription(subId, "logs", allFilter))

	for currentIndex, currentTopic = range topics {
		eventData := ptypes.EventData{
			Topics: []string{strconv.Itoa(currentIndex)},
		}
		message, err = proto.Marshal(&eventData)
		require.NoError(t, err)
		ethSubSet.Reset()
		ethSubSet.Publish(pubsub.NewMessage(string(message), message))
		require.Equal(t, messageShouldBeSent[currentIndex], messageSent)
		messageSent = false
	}

	// If don't call Reset() then all should fail as does not repeat
	// sending to same address.
	for currentIndex, currentTopic = range topics {
		message = []byte(strconv.Itoa(currentIndex))
		ethSubSet.Publish(pubsub.NewMessage(string(message), message))
		require.Equal(t, false, messageSent)
		messageSent = false
	}

	conn.connected = false
	currentIndex = 1
	currentTopic = topics[currentIndex]

	eventData := ptypes.EventData{
		Topics: []string{strconv.Itoa(currentIndex)},
	}
	message, err = proto.Marshal(&eventData)
	require.NoError(t, err)
	ethSubSet.Reset()
	ethSubSet.Publish(pubsub.NewMessage(string(message), message))
	require.True(t, messageSent)
	messageSent = false

	// Need to wait long enough for purge go routine to run
	// time.Sleep(400000)
	ethSubSet.Reset()
	ethSubSet.Publish(pubsub.NewMessage(string(message), message))
	// require.False(t, messageSent)
}

func testEthWriter(t *testing.T, conn *mockConnection, id string, subs *LegacyEthSubscriptionSet) pubsub.SubscriberFunc {
	return func(msg pubsub.Message) {
		defer func() {
			if r := recover(); r != nil {
				require.False(t, conn.connected)
				go subs.Purge(conn.caller)
			} else {
				require.True(t, conn.connected)
			}
		}()
		resp := rpctypes.RPCResponse{
			JSONRPC: "2.0",
			ID:      rpctypes.JSONRPCStringID(id),
		}
		ethMsg := ptypes.EthMessage{}
		require.NoError(t, proto.Unmarshal(msg.Body(), &ethMsg), "unmarshall message in callback")
		resp.Result = ethMsg.Body
		messageSent = true

		require.True(t, messageShouldBeSent[currentIndex], "topic should not match")
		require.True(t, 0 == bytes.Compare(message, resp.Result), "message sent")
		require.Equal(t, subId, fmt.Sprintf("%s", resp.ID), "id sent")

		if !conn.connected {
			panic("caller is not connectede")
		}
	}
}

type mockConnection struct {
	caller    string
	connected bool
}
