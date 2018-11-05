package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cretz/bine/tor"
	"golang.org/x/net/proxy"
)

// TestIsolateSocksAuth simply confirms the functionality of IsolateSOCKSAuth,
// namely that it uses a different circuit for different SOCKS credentials.
func TestIsolateSocksAuth(t *testing.T) {
	// Create context w/ no isolate
	ctx := NewTestContext(t, &tor.StartConf{
		NoAutoSocksPort: true,
		ExtraArgs:       []string{"--SocksPort", "auto NoIsolateSOCKSAuth"},
	})
	// Make sure it reused the circuit (i.e. only has one) for both separate-auth calls
	uniqueCircuitIDs := doSeparateAuthHttpCalls(ctx)
	ctx.Debugf("Unique IDs without isolate: %v", uniqueCircuitIDs)
	ctx.Require.Len(uniqueCircuitIDs, 1)
	// Create context w/ isolate
	ctx = NewTestContext(t, nil)
	// Make sure it made a new circuit (i.e. has two) for each separate-auth call
	uniqueCircuitIDs = doSeparateAuthHttpCalls(ctx)
	ctx.Debugf("Unique IDs with isolate: %v", uniqueCircuitIDs)
	ctx.Require.Len(uniqueCircuitIDs, 2)
}

// Returns the map keyed with unique circuit IDs after second separate-auth HTTP call
func doSeparateAuthHttpCalls(ctx *TestContext) map[int]struct{} {
	defer ctx.Close()
	enableCtx, enableCancel := context.WithTimeout(ctx, 100*time.Second)
	defer enableCancel()
	ctx.Require.NoError(ctx.EnableNetwork(enableCtx, true))
	// Make HTTP call w/ an auth
	client := httpClient(ctx, &tor.DialConf{ProxyAuth: &proxy.Auth{"foo", "bar"}})
	byts := httpGet(ctx, client, "https://check.torproject.org/api/ip")
	ctx.Debugf("Read bytes: %v", string(byts))
	// Confirm just size 1
	ids := uniqueStreamCircuitIDs(ctx)
	ctx.Require.Len(ids, 1)
	// Now make call with another auth and just return circuit IDs
	client = httpClient(ctx, &tor.DialConf{ProxyAuth: &proxy.Auth{"baz", "qux"}})
	byts = httpGet(ctx, client, "https://check.torproject.org/api/ip")
	ctx.Debugf("Read bytes: %v", string(byts))
	return uniqueStreamCircuitIDs(ctx)
}

// Return each stream circuit as a key of an empty-val map
func uniqueStreamCircuitIDs(ctx *TestContext) map[int]struct{} {
	ret := map[int]struct{}{}
	vals, err := ctx.Control.GetInfo("stream-status")
	ctx.Require.NoError(err)
	for _, val := range vals {
		ctx.Require.Equal("stream-status", val.Key)
		for _, line := range strings.Split(val.Val, "\n") {
			pieces := strings.Split(strings.TrimSpace(line), " ")
			if len(pieces) < 3 {
				continue
			}
			i, err := strconv.Atoi(pieces[2])
			ctx.Require.NoError(err)
			ret[i] = struct{}{}
		}
	}
	return ret
}
