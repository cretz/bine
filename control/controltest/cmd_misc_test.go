package controltest

import "testing"

func TestSignal(t *testing.T) {
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	ctx.Require.NoError(conn.Signal("HEARTBEAT"))
}
