package controltest

import (
	"context"
	"testing"
	"time"

	"github.com/cretz/bine/control"
)

func TestEvents(t *testing.T) {
	SkipIfNotRunningSpecifically(t)
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	// Turn on event handler
	eventCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()
	go func() { ctx.Require.Equal(context.Canceled, conn.HandleEvents(eventCtx)) }()
	// Enable all events and hold on to which ones were seen
	allEvents := control.EventCodes()
	seenEvents := map[control.EventCode]struct{}{}
	ch := make(chan control.Event, 1000)
	ctx.Require.NoError(conn.AddEventListener(ch, allEvents...))
	// Turn on the network
	ctx.Require.NoError(conn.SetConf(control.NewKeyVal("DisableNetwork", "0")))
MainLoop:
	for {
		select {
		case e := <-ch:
			// Remove the event listener if it was seen
			if _, ok := seenEvents[e.Code()]; !ok {
				ctx.Debugf("Got event: %v", e.Code())
				seenEvents[e.Code()] = struct{}{}
				ctx.Require.NoError(conn.RemoveEventListener(ch, e.Code()))
			}
		case <-time.After(3 * time.Second):
			ctx.Debugf("3 seconds passed")
			break MainLoop
		}
	}
	// Check that each event was sent at least once
	for _, event := range allEvents {
		_, ok := seenEvents[event]
		ctx.Debugf("Event %v seen? %v", event, ok)
	}
}
