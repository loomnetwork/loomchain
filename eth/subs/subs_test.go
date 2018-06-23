// +build evm

package subs

import (
	"bytes"
	"github.com/loomnetwork/loomchain/eth/phonkee/go-pubsub"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/rpc/lib/types"
	"strconv"
	"testing"
)

const (
	testFilter = "{\"address\":\"0x8888F1F195AfA192cFee860698584C030f4c9Db1\",\"topics\":[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",null,[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",\"0x0000000000000000000000000aff3454fce5edbc8cca8697c15331677e6ebccc\"]]}"
	noneFilter = "{\"address\":\"\",\"topics\":[]}"
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
	messageShouldBeSent = []bool{false, true, true, true}
	messageSent         = false
)

func TestTopicsFromFilter(t *testing.T) {
	topics, err := topicsFromFilter(noneFilter)
	require.NoError(t, err)
	require.Equal(t, 0, len(topics), "should be no topics")
	topics, err = topicsFromFilter(testFilter)
	require.NoError(t, err)
	require.Equal(t, 4, len(topics), "wrong number of topics")
}

func TestUnSubscribe(t *testing.T) {
	ethSubSet := NewEthSubscriptionSet()
	conn := mockConnection{
		caller:    "myCaller",
		connected: true,
	}

	var sub pubsub.Subscriber
	sub, subId = ethSubSet.For(conn.caller)
	sub.Do(testEthWriter(t, &conn, subId, ethSubSet))
	ethSubSet.AddSubscription(subId, "logs", testFilter)

	currentIndex = 1
	currentTopic = topics[currentIndex]
	message = []byte(strconv.Itoa(currentIndex))
	ethSubSet.Publish(pubsub.NewMessage(currentTopic, message))
	require.True(t, messageSent)
	messageSent = false

	ethSubSet.Remove(subId)
	ethSubSet.Publish(pubsub.NewMessage(currentTopic, message))
	require.False(t, messageSent)
	messageSent = false
}

func TestSubscribe(t *testing.T) {
	ethSubSet := NewEthSubscriptionSet()
	conn := mockConnection{
		caller:    "myCaller",
		connected: true,
	}

	var sub pubsub.Subscriber
	sub, subId = ethSubSet.For(conn.caller)
	sub.Do(testEthWriter(t, &conn, subId, ethSubSet))
	ethSubSet.AddSubscription(subId, "logs", testFilter)

	for currentIndex, currentTopic = range topics {
		message = []byte(strconv.Itoa(currentIndex))
		ethSubSet.Reset()
		ethSubSet.Publish(pubsub.NewMessage(currentTopic, message))
		require.Equal(t, messageShouldBeSent[currentIndex], messageSent)
		messageSent = false
	}

	// If don't call Reset() then all should fail as does not repat
	// sending to same address.
	for currentIndex, currentTopic = range topics {
		message = []byte(strconv.Itoa(currentIndex))
		ethSubSet.Publish(pubsub.NewMessage(currentTopic, message))
		require.Equal(t, false, messageSent)
		messageSent = false
	}

	conn.connected = false
	currentIndex = 1
	currentTopic = topics[currentIndex]
	message = []byte(strconv.Itoa(currentIndex))
	ethSubSet.Reset()
	ethSubSet.Publish(pubsub.NewMessage(currentTopic, message))
	require.True(t, messageSent)
	messageSent = false

	// Need to wait long enough for purge go routine to run
	// time.Sleep(400000)
	ethSubSet.Reset()
	ethSubSet.Publish(pubsub.NewMessage(currentTopic, message))
	// require.False(t, messageSent)
}

func testEthWriter(t *testing.T, conn *mockConnection, id string, subs *EthSubscriptionSet) pubsub.SubscriberFunc {
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
			ID:      id,
		}
		resp.Result = msg.Body()
		messageSent = true
		require.True(t, messageShouldBeSent[currentIndex], "topic should not match")
		require.Equal(t, currentTopic, msg.Topic(), "wrong topic matched")
		require.True(t, 0 == bytes.Compare(message, resp.Result), "message sent")
		require.Equal(t, subId, resp.ID, "id sent")

		if !conn.connected {
			panic("caller is not connectede")
		}
	}
}

type mockConnection struct {
	caller    string
	connected bool
}
