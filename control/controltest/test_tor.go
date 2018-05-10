package controltest

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cretz/bine/process"
)

var torExePath string

func init() {
	flag.StringVar(&torExePath, "tor", "tor", "The TOR exe path")
	flag.Parse()
}

type TestTor struct {
	DataDir     string
	OrigArgs    []string
	ControlPort int

	processCancelFn context.CancelFunc
}

func StartTestTor(ctx context.Context, extraArgs ...string) (*TestTor, error) {
	dataDir, err := ioutil.TempDir(".", "test-data-dir-")
	if err != nil {
		return nil, err
	}
	controlPortFile := filepath.Join(dataDir, "control-port")
	ret := &TestTor{
		DataDir: dataDir,
		OrigArgs: append([]string{
			// "--quiet",
			"--DisableNetwork", "1",
			"--ControlPort", "auto",
			"--ControlPortWriteToFile", controlPortFile,
			"--DataDirectory", dataDir,
		}, extraArgs...),
	}
	errCh := make(chan error, 1)
	var processCtx context.Context
	processCtx, ret.processCancelFn = context.WithCancel(ctx)
	go func() {
		p, err := process.NewProcessCreator(torExePath).New(processCtx, ret.OrigArgs...)
		if err == nil {
			err = p.Run()
		}
		errCh <- err
	}()
	err = nil
	for err == nil {
		select {
		case err = <-errCh:
			if err == nil {
				err = fmt.Errorf("Process returned earlier than expected")
			}
		case <-processCtx.Done():
			err = ctx.Err()
		default:
			// Try to read the controlport file, or wait a bit
			var byts []byte
			if byts, err = ioutil.ReadFile(controlPortFile); err == nil {
				if ret.ControlPort, err = process.ControlPortFromFileContents(string(byts)); err == nil {
					return ret, nil
				}
			} else if os.IsNotExist(err) {
				// Wait a bit
				err = nil
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	// Delete the data dir and stop the process since we errored
	if closeErr := ret.Close(); closeErr != nil {
		fmt.Printf("Warning, unable to remove data dir %v: %v", dataDir, closeErr)
	}
	return nil, err
}

func (t *TestTor) Close() (err error) {
	if t.processCancelFn != nil {
		t.processCancelFn()
	}
	// Try this twice while waiting a bit between each
	for i := 0; i < 2; i++ {
		if err = os.RemoveAll(t.DataDir); err == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	return
}
