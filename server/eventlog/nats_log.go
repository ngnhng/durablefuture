package eventlog

import (
	_ "github.com/DeluxeOwl/chronicle/event"
	"github.com/DeluxeOwl/chronicle/eventlog"
)

func test() {
	eventlog.NewNATSJetStream(nil)
}
