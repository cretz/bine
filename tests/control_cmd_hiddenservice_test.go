package tests

import (
	"context"
	"testing"
	"time"

	"github.com/cretz/bine/control"
)

const hsFetchOnion = "2gzyxa5ihm7nsggfxnu52rck2vv4rvmdlkiu3zzui5du4xyclen53wid"

func TestHSFetch(t *testing.T) {
	ctx := GlobalEnabledNetworkContext(t)
	// Add listener
	eventCh := make(chan control.Event)
	defer close(eventCh)
	err := ctx.Control.AddEventListener(eventCh, control.EventCodeHSDescContent)
	ctx.Require.NoError(err)
	defer ctx.Control.RemoveEventListener(eventCh, control.EventCodeHSDescContent)
	// Lookup HS
	err = ctx.Control.GetHiddenServiceDescriptorAsync(hsFetchOnion, "")
	ctx.Require.NoError(err)
	// Grab events
	eventCtx, eventCancel := context.WithTimeout(ctx, 45*time.Second)
	defer eventCancel()
	errCh := make(chan error, 1)
	go func() { errCh <- ctx.Control.HandleEvents(eventCtx) }()
	select {
	case <-eventCtx.Done():
		ctx.Require.NoError(eventCtx.Err())
	case err := <-errCh:
		ctx.Require.NoError(err)
	case event := <-eventCh:
		hsEvent := event.(*control.HSDescContentEvent)
		ctx.Require.Equal(hsFetchOnion, hsEvent.Address)
	}
}
