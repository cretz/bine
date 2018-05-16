package tests

import (
	"strings"
	"testing"
)

func TestProtocolInfo(t *testing.T) {
	ctx := NewTestContext(t, nil)
	defer ctx.Close()
	info, err := ctx.Control.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.Contains(info.AuthMethods, "SAFECOOKIE")
	ctx.Require.True(strings.HasPrefix(info.TorVersion, "0.3"))
}
