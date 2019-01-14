package events

import (
	"encoding/json"

	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/log"
)

// LogEventDispatcher just logs events
type LogEventDispatcher struct {
}

// NewLogEventDispatcher create a new redis dispatcher
func NewLogEventDispatcher() *LogEventDispatcher {
	return &LogEventDispatcher{}
}

// Send sends the event
func (ed *LogEventDispatcher) Send(index uint64, msg []byte) error {
	var logs ptypes.EventData
	//Can use proto.Unmarshal as well but it required proto.Marshal in following as well - emitMsg, err := json.Marshal(&eventData) in event.go, This was causing incorrect formatting of some logs
	//Following code in log.Debug("Received published message", "msg", msg.Body(), "remote", clientCtx.GetRemoteAddr()) in query_server.go so used JSON.marshal but JSON it produces for published message is escaped json.
	//Fix for the same is used in this function below
	if err := json.Unmarshal(msg, &logs); err != nil {
		return err
	}
	//NewJSONLogger is used of go-kit, this is primarily to avoid escaped JSON and proper JSON  format
	log.JSONInfo("Event emitted", "index", index, "length", len(msg), "msg", logs)

	return nil
}
