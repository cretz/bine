package controltest

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	ExtraTorArgs []string
	Require      *require.Assertions
	TestTor      *TestTor
	DebugWriter  io.Writer
}

func NewTestContext(ctx context.Context, t *testing.T, extraTorArgs ...string) *TestContext {
	ret := &TestContext{Context: ctx, T: t, ExtraTorArgs: extraTorArgs, Require: require.New(t)}
	testVerboseFlag := flag.Lookup("test.v")
	if testVerboseFlag != nil && testVerboseFlag.Value != nil && testVerboseFlag.Value.String() == "true" {
		ret.DebugWriter = os.Stdout
	} else {
		ret.ExtraTorArgs = append(append([]string{}, ret.ExtraTorArgs...), "--quiet")
	}
	return ret
}

func NewTestContextConnected(t *testing.T, extraTorArgs ...string) (*TestContext, *control.Conn) {
	ctx := NewTestContext(context.Background(), t, extraTorArgs...)
	conn, err := ctx.ConnectTestTor()
	if err != nil {
		ctx.Close()
		ctx.Fatal(err)
	}
	return ctx, conn
}

func NewTestContextAuthenticated(t *testing.T, extraTorArgs ...string) (*TestContext, *control.Conn) {
	ctx, conn := NewTestContextConnected(t, extraTorArgs...)
	if err := conn.Authenticate(""); err != nil {
		conn.Close()
		ctx.Close()
		ctx.Fatal(err)
	}
	return ctx, conn
}

func (t *TestContext) EnsureTestTorStarted() {
	if t.TestTor == nil {
		var err error
		if t.TestTor, err = StartTestTor(t, t.ExtraTorArgs...); err != nil {
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

func (t *TestContext) CloseConnected(conn *control.Conn) {
	if err := conn.Close(); err != nil {
		fmt.Printf("Warning, close failed on tor conn: %v", err)
	}
	t.Close()
}

func (t *TestContext) ConnectTestTor() (*control.Conn, error) {
	t.EnsureTestTorStarted()
	textConn, err := textproto.Dial("tcp", "127.0.0.1:"+strconv.Itoa(t.TestTor.ControlPort))
	if err != nil {
		return nil, err
	}
	conn := control.NewConn(textConn)
	conn.DebugWriter = t.DebugWriter
	return conn, nil
}

func (t *TestContext) DebugEnabled() bool {
	return t.DebugWriter != nil
}

func (t *TestContext) Debugf(format string, args ...interface{}) {
	if w := t.DebugWriter; w != nil {
		fmt.Fprintf(w, format+"\n", args...)
	}
}
