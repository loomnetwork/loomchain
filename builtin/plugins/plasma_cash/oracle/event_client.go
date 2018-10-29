package oracle

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	loom_client "github.com/loomnetwork/go-loom/client"
	lptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/phonkee/go-pubsub"
	"github.com/pkg/errors"
)

const (
	SubmitBlockConfirmedEventTopic = "pcash:submitblockconfirmed"
	ExitConfirmedEventTopic        = "pcash:exitconfirmed"
	WithdrawConfirmedEventTopic    = "pcash:withdrawconfirmed"
	ResetConfirmedEventTopic       = "pcash:resetconfirmed"
	DepositConfirmedEventTopic     = "pcash:depositconfirmed"

	DefaultPingInterval         = 10 * time.Second
	DefaultPingDeadlineDuration = 10 * time.Second
)

type DAppChainEventClient struct {
	ws                 *websocket.Conn
	nextMsgID          uint64
	chainEventQuitCh   chan struct{}
	chainEventSubCount int
	chainEventHub      pubsub.Hub

	Address loom.Address
}

func ConnectToDAppChainGateway(contractAddr loom.Address, eventsURI string) (*DAppChainEventClient, error) {
	var ws *websocket.Conn

	if eventsURI == "" {
		return nil, fmt.Errorf("event uri cannot be empty")
	}

	ws, _, err := websocket.DefaultDialer.Dial(eventsURI, nil)
	if err != nil {
		return nil, err
	}

	return &DAppChainEventClient{
		ws:      ws,
		Address: contractAddr,
	}, nil
}

type EventSub struct {
	subscriber pubsub.Subscriber
	closeFn    func()
}

func (es *EventSub) Close() {
	es.subscriber.Close()
	es.closeFn()
}

func (d *DAppChainEventClient) WatchWithdrawConfirmedEvent(
	sink chan<- *ptypes.PlasmaCashWithdrawConfirmedEvent) (*EventSub, error) {
	if d.ws == nil {
		return nil, errors.New("websocket events unavailable")
	}

	if err := d.subChainEvents(); err != nil {
		return nil, err
	}

	sub := d.chainEventHub.Subscribe("event")
	sub.Do(func(msg pubsub.Message) {
		ev := lptypes.EventData{}
		if err := proto.Unmarshal(msg.Body(), &ev); err != nil {
			return
		}
		if ev.Topics == nil || ev.Topics[0] != WithdrawConfirmedEventTopic {
			return
		}
		contractAddr := loom.UnmarshalAddressPB(ev.Address)
		if contractAddr.Compare(d.Address) != 0 {
			return
		}
		payload := ptypes.PlasmaCashWithdrawConfirmedEvent{}
		if err := proto.Unmarshal(ev.EncodedBody, &payload); err != nil {
			return
		}
		sink <- &payload
	})

	return &EventSub{
		subscriber: sub,
		closeFn:    d.unsubChainEvents,
	}, nil
}

func (d *DAppChainEventClient) WatchExitConfirmedEvent(
	sink chan<- *ptypes.PlasmaCashExitConfirmedEvent) (*EventSub, error) {
	if d.ws == nil {
		return nil, errors.New("websocket events unavailable")
	}

	if err := d.subChainEvents(); err != nil {
		return nil, err
	}

	sub := d.chainEventHub.Subscribe("event")
	sub.Do(func(msg pubsub.Message) {
		ev := lptypes.EventData{}
		if err := proto.Unmarshal(msg.Body(), &ev); err != nil {
			return
		}
		if ev.Topics == nil || ev.Topics[0] != ExitConfirmedEventTopic {
			return
		}
		contractAddr := loom.UnmarshalAddressPB(ev.Address)
		if contractAddr.Compare(d.Address) != 0 {
			return
		}
		payload := ptypes.PlasmaCashExitConfirmedEvent{}
		if err := proto.Unmarshal(ev.EncodedBody, &payload); err != nil {
			return
		}
		sink <- &payload
	})

	return &EventSub{
		subscriber: sub,
		closeFn:    d.unsubChainEvents,
	}, nil
}

func (d *DAppChainEventClient) WatchDepositConfirmedEvent(
	sink chan<- *ptypes.PlasmaCashDepositConfirmedEvent) (*EventSub, error) {
	if d.ws == nil {
		return nil, errors.New("websocket events unavailable")
	}

	if err := d.subChainEvents(); err != nil {
		return nil, err
	}

	sub := d.chainEventHub.Subscribe("event")
	sub.Do(func(msg pubsub.Message) {
		ev := lptypes.EventData{}
		if err := proto.Unmarshal(msg.Body(), &ev); err != nil {
			return
		}
		if ev.Topics == nil || ev.Topics[0] != DepositConfirmedEventTopic {
			return
		}
		contractAddr := loom.UnmarshalAddressPB(ev.Address)
		if contractAddr.Compare(d.Address) != 0 {
			return
		}
		payload := ptypes.PlasmaCashDepositConfirmedEvent{}
		if err := proto.Unmarshal(ev.EncodedBody, &payload); err != nil {
			return
		}
		sink <- &payload
	})

	return &EventSub{
		subscriber: sub,
		closeFn:    d.unsubChainEvents,
	}, nil
}

func (d *DAppChainEventClient) WatchSubmitBlockConfirmedEvent(
	sink chan<- *ptypes.PlasmaCashSubmitBlockConfirmedEvent) (*EventSub, error) {
	if d.ws == nil {
		return nil, errors.New("websocket events unavailable")
	}

	if err := d.subChainEvents(); err != nil {
		return nil, err
	}

	sub := d.chainEventHub.Subscribe("event")
	sub.Do(func(msg pubsub.Message) {
		ev := lptypes.EventData{}
		if err := proto.Unmarshal(msg.Body(), &ev); err != nil {
			return
		}
		if ev.Topics == nil || ev.Topics[0] != SubmitBlockConfirmedEventTopic {
			return
		}
		contractAddr := loom.UnmarshalAddressPB(ev.Address)
		if contractAddr.Compare(d.Address) != 0 {
			return
		}
		payload := ptypes.PlasmaCashSubmitBlockConfirmedEvent{}
		if err := proto.Unmarshal(ev.EncodedBody, &payload); err != nil {
			return
		}
		sink <- &payload
	})

	return &EventSub{
		subscriber: sub,
		closeFn:    d.unsubChainEvents,
	}, nil
}

func (d *DAppChainEventClient) subChainEvents() error {
	d.chainEventSubCount++
	if d.chainEventSubCount > 1 {
		return nil // already subscribed
	}

	err := d.ws.WriteJSON(&loom_client.RPCRequest{
		Version: "2.0",
		Method:  "subevents",
		ID:      strconv.FormatUint(d.nextMsgID, 10),
	})
	d.nextMsgID++

	if err != nil {
		return errors.Wrap(err, "failed to subscribe to DAppChain events")
	}

	resp := loom_client.RPCResponse{}
	if err = d.ws.ReadJSON(&resp); err != nil {
		return errors.Wrap(err, "failed to subscribe to DAppChain events")
	}
	if resp.Error != nil {
		return errors.Wrap(resp.Error, "failed to subscribe to DAppChain events")
	}

	d.chainEventHub = pubsub.New()
	d.chainEventQuitCh = make(chan struct{})

	go pumpChainEvents(d.ws, d.chainEventHub, d.chainEventQuitCh)

	return nil
}

func (d *DAppChainEventClient) unsubChainEvents() {
	d.chainEventSubCount--
	if d.chainEventSubCount > 0 {
		return // still have subscribers
	}

	close(d.chainEventQuitCh)

	d.ws.WriteJSON(&loom_client.RPCRequest{
		Version: "2.0",
		Method:  "unsubevents",
		ID:      strconv.FormatUint(d.nextMsgID, 10),
	})
	d.nextMsgID++
}

func pumpChainEvents(ws *websocket.Conn, hub pubsub.Hub, quit <-chan struct{}) {
	pingTimer := time.NewTimer(DefaultPingInterval)
	for {
		select {
		case <-pingTimer.C:
			if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(DefaultPingDeadlineDuration)); err != nil {
				fmt.Printf("error in sending a ping, will retry in next iteration")
			}
			break
		case <-quit:
			return
		default:
			resp := loom_client.RPCResponse{}
			if err := ws.ReadJSON(&resp); err != nil {
				panic(err)
			}
			if resp.Error != nil {
				panic(resp.Error)
			}
			unmarshaller := jsonpb.Unmarshaler{}
			reader := bytes.NewBuffer(resp.Result)
			eventData := lptypes.EventData{}
			if err := unmarshaller.Unmarshal(reader, &eventData); err != nil {
				panic(err)
			}
			bytes, err := proto.Marshal(&eventData)
			if err != nil {
				panic(err)
			}
			hub.Publish(pubsub.NewMessage("event", bytes))
		}
	}
}
