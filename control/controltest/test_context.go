package controltest

import (
	"context"
	"fmt"
	"net/textproto"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cretz/bine/control"
)

type TestContext struct {
	context.Context
	*testing.T
	Require *require.Assertions
	TestTor *TestTor
}

func NewTestContext(ctx context.Context, t *testing.T) *TestContext {
	return &TestContext{Context: ctx, T: t, Require: require.New(t)}
}

func (t *TestContext) EnsureTestTorStarted() {
	if t.TestTor == nil {
		var err error
		if t.TestTor, err = StartTestTor(t); err != nil {
			t.Fatal(err)
		}
	}
}

func (t *TestContext) Close() {
	if t.TestTor != nil {
		if err := t.TestTor.Close(); err != nil {
			fmt.Printf("Warning, close failed on tor inst: %v", err)
		}
	}
}

func (t *TestContext) ConnectTestTor() *control.Conn {
	t.EnsureTestTorStarted()
	textConn, err := textproto.Dial("tcp", "127.0.0.1:"+strconv.Itoa(t.TestTor.ControlPort))
	if err != nil {
		t.Fatal(err)
	}
	conn := control.NewConn(textConn)
	conn.DebugWriter = os.Stdout
	return conn
}
