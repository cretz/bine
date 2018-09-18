package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cretz/bine/control"
)

func TestHSFetch(t *testing.T) {
	ctx := GlobalEnabledNetworkContext(t)
	// Add listener
	eventCh := make(chan control.Event)
	defer close(eventCh)
	err := ctx.Control.AddEventListener(eventCh, control.EventCodeHSDescContent)
	ctx.Require.NoError(err)
	defer ctx.Control.RemoveEventListener(eventCh, control.EventCodeHSDescContent)
	// Lookup HS
	err = ctx.Control.GetHiddenServiceDescriptorAsync("facebookcorewwwi", "")
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
		ctx.Require.Equal("facebookcorewwwi", hsEvent.Address)
		ctx.Require.True(strings.HasPrefix(hsEvent.Descriptor, "rendezvous-service-descriptor "+hsEvent.DescID))
	}
}
