package controltest

import (
	"strings"
	"testing"
)

func TestProtocolInfo(t *testing.T) {
	ctx, conn := NewTestContextConnected(t)
	defer ctx.CloseConnected(conn)
	info, err := conn.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.Contains(info.AuthMethods, "NULL")
	ctx.Require.True(strings.HasPrefix(info.TorVersion, "0.3"))
}
