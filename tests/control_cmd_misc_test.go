package tests

import "testing"

func TestSignal(t *testing.T) {
	ctx := NewTestContext(t, nil)
	defer ctx.Close()
	ctx.Require.NoError(ctx.Control.Signal("HEARTBEAT"))
}
