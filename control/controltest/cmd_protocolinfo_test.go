package controltest

import (
	"context"
	"strings"
	"testing"
)

func TestProtocolInfo(t *testing.T) {
	ctx := NewTestContext(context.Background(), t)
	defer ctx.Close()
	conn := ctx.ConnectTestTor()
	defer conn.Close()
	info, err := conn.RequestProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.Contains(info.AuthMethods, "NULL")
	ctx.Require.True(strings.HasPrefix(info.TorVersion, "0.3"))
}
