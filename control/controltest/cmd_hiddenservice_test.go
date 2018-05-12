package controltest

import (
	"testing"
)

func TestGetHiddenServiceDescriptorAsync(t *testing.T) {
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	t.Skip("TODO")
}
