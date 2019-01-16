package events

import (
	"encoding/json"
	"io"

	kitlog "github.com/go-kit/kit/log"
	loom "github.com/loomnetwork/go-loom"
	etypes "github.com/loomnetwork/go-loom/plugin/types"
)

// LogEventDispatcher just logs events
type LogEventDispatcher struct {
	eventdispatch string
	JSONLogger    *loom.Logger
}

var logevent LogEventDispatcher

func InitEventLogConfig(loglevel string, logdst string, eventdispatch string) error {
	logevent.eventdispatch = eventdispatch
	w := loom.MakeFileLoggerWriter(loglevel, logdst)
	jlogTr := func(w io.Writer) kitlog.Logger {
		return kitlog.NewJSONLogger(w)
	}
	//JSONLogger instance can be created as and when it is needed
	logevent.JSONLogger = loom.MakeLoomLogger(loglevel, w, jlogTr)

	return nil
}

func NoopLogEventDispatcher() *LogEventDispatcher {
	var eventdispatcher *LogEventDispatcher
	return eventdispatcher
}

// NewLogEventDispatcher create a new redis dispatcher
func NewLogEventDispatcher() *LogEventDispatcher {

	return &LogEventDispatcher{}
}

// Send sends the event
func (ed *LogEventDispatcher) Send(index uint64, msg []byte) error {
	var logs etypes.EventData
	//Can use proto.Unmarshal as well but it required proto.Marshal in following as well - emitMsg, err := json.Marshal(&eventData) in event.go, This was causing incorrect formatting of some logs
	//Following code in log.Debug("Received published message", "msg", msg.Body(), "remote", clientCtx.GetRemoteAddr()) in query_server.go so used JSON.marshal but JSON it produces for published message is escaped json.
	//Fix for the same is used in this function below
	if err := json.Unmarshal(msg, &logs); err != nil {
		return err
	}
	if logevent.eventdispatch == "log" {

		logevent.JSONLogger.Info("Event emitted", "index", index, "length", len(msg), "msg", logs)

	}

	return nil
}
