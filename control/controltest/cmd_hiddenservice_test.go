package controltest

import (
	"testing"
	"time"

	"github.com/cretz/bine/control"
)

func TestGetHiddenServiceDescriptorAsync(t *testing.T) {
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	// Enable the network
	ctx.Require.NoError(conn.SetConf(control.NewKeyVal("DisableNetwork", "0")))
	ctx.Require.NoError(conn.GetHiddenServiceDescriptorAsync("facebookcorewwwi", ""))
	time.Sleep(60 * time.Second)
}
