package tests

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/stretchr/testify/require"
)

var torEnabled bool
var torExePath string
var torVerbose bool
var torIncludeNetworkTests bool
var globalEnabledNetworkContext *TestContext

func TestMain(m *testing.M) {
	flag.BoolVar(&torEnabled, "tor", false, "Whether any of the integration tests are enabled")
	flag.StringVar(&torExePath, "tor.path", "tor", "The Tor exe path")
	flag.BoolVar(&torVerbose, "tor.verbose", false, "Show verbose test info")
	flag.BoolVar(&torIncludeNetworkTests, "tor.network", false, "Include network tests")
	flag.Parse()
	exitCode := m.Run()
	if globalEnabledNetworkContext != nil {
		globalEnabledNetworkContext.CloseTorOnClose = true
		globalEnabledNetworkContext.Close()
	}
	os.Exit(exitCode)
}

func GlobalEnabledNetworkContext(t *testing.T) *TestContext {
	if !torEnabled || !torIncludeNetworkTests {
		t.Skip("Only runs if -tor and -tor.network are set")
	}
	if globalEnabledNetworkContext == nil {
		ctx := NewTestContext(t, nil)
		ctx.CloseTorOnClose = false
		// 45 second wait for enable network
		enableCtx, enableCancel := context.WithTimeout(ctx, 45*time.Second)
		defer enableCancel()
		ctx.Require.NoError(ctx.EnableNetwork(enableCtx, true))
		globalEnabledNetworkContext = ctx
	} else {
		globalEnabledNetworkContext.T = t
		globalEnabledNetworkContext.Require = require.New(t)
	}
	return globalEnabledNetworkContext
}

type TestContext struct {
	context.Context
	*testing.T
	*tor.Tor
	Require         *require.Assertions
	CloseTorOnClose bool
}

func NewTestContext(t *testing.T, conf *tor.StartConf) *TestContext {
	if !torEnabled {
		t.Skip("Only runs if -tor is set")
	}
	// Build start conf
	if conf == nil {
		conf = &tor.StartConf{}
	}
	conf.ExePath = torExePath
	if torVerbose {
		conf.DebugWriter = os.Stdout
		conf.NoHush = true
	} else {
		conf.ExtraArgs = append(conf.ExtraArgs, "--quiet")
	}
	ret := &TestContext{Context: context.Background(), T: t, Require: require.New(t), CloseTorOnClose: true}
	// Start tor
	var err error
	if ret.Tor, err = tor.Start(ret.Context, conf); err != nil {
		defer ret.Close()
		t.Fatal(err)
	}
	return ret
}

func (t *TestContext) Close() {
	if t.CloseTorOnClose {
		if err := t.Tor.Close(); err != nil {
			if t.Failed() {
				t.Logf("Failure on close: %v", err)
			} else {
				t.Errorf("Failure on close: %v", err)
			}
		}
	}
}
