package controltest

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/cretz/bine/tor"
	"github.com/stretchr/testify/require"
)

type TestContext struct {
	context.Context
	*testing.T
	*tor.Tor
	Require *require.Assertions
}

var torExePath string

func init() {
	flag.StringVar(&torExePath, "tor.path", "tor", "The TOR exe path")
	flag.Parse()
}

func NewTestContext(t *testing.T, conf *tor.StartConf) *TestContext {
	// Build start conf
	if conf == nil {
		conf = &tor.StartConf{}
	}
	conf.ExePath = torExePath
	if f := flag.Lookup("test.v"); f != nil && f.Value != nil && f.Value.String() == "true" {
		conf.DebugWriter = os.Stdout
	} else {
		conf.ExtraArgs = append(conf.ExtraArgs, "--quiet")
	}
	ret := &TestContext{Context: context.Background(), T: t, Require: require.New(t)}
	// Start tor
	var err error
	if ret.Tor, err = tor.Start(ret.Context, conf); err != nil {
		defer ret.Close()
		t.Fatal(err)
	}
	return ret
}

func (t *TestContext) Close() {
	if err := t.Tor.Close(); err != nil {
		if t.Failed() {
			t.Logf("Failure on close: %v", err)
		} else {
			t.Errorf("Failure on close: %v", err)
		}
	}
}
