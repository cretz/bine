package tor

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/textproto"
	"os"
	"strconv"
	"time"

	"github.com/cretz/bine/control"

	"github.com/cretz/bine/process"
)

type Tor struct {
	Process process.Process
	Control *control.Conn

	ProcessCancelFunc    context.CancelFunc
	ControlPort          int
	DataDir              string
	DeleteDataDirOnClose bool
	DebugWriter          io.Writer
}

type StartConf struct {
	// TODO: docs...Empty string means just "tor" either locally or on PATH
	ExePath string
	// TODO: docs...If true, doesn't use exe path, uses statically compiled Tor
	Embedded bool
	// TODO: docs...If 0, Tor is asked to store the control port in a temporary file in the data directory
	ControlPort int
	//  TODO: docs...If not empty, this is the data directory used and *TempDataDir* fields are unused
	DataDir string
	// TODO: docs...by default we do cookie auth, this disables it
	DisableCookieAuth bool
	// TODO: docs...by default this authenticates
	DisableEagerAuth bool
	// TODO: docs...by default network is disabled
	EnableNetwork bool
	//  TODO: docs...If not empty, this is the parent directory that a child dir is created for data. If empty, the
	// current dir is assumed. This has no effect if DataDir is set.
	TempDataDirBase string
	//  TODO: docs...If true the temporary data dir is not deleted on close. This has no effect if DataDir is set.
	RetainTempDataDir bool
	//  TODO: docs...Any extra CLI arguments to pass to Tor. This are applied after other CLI args.
	ExtraArgs []string
	// TODO: docs...If not present, a blank torrc file is placed in the data dir and used
	TorrcFile string
	// TODO: docs...
	DebugWriter io.Writer
}

// TODO: docs...conf can be nil for defaults, note on error the process could still be running
func Start(ctx context.Context, conf *StartConf) (*Tor, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if conf == nil {
		conf = &StartConf{}
	}
	tor := &Tor{DataDir: conf.DataDir, DebugWriter: conf.DebugWriter}
	// Create the data dir
	if tor.DataDir == "" {
		tempBase := conf.TempDataDirBase
		if tempBase == "" {
			tempBase = "."
		}
		var err error
		if tor.DataDir, err = ioutil.TempDir(tempBase, "data-dir-"); err != nil {
			return nil, fmt.Errorf("Unable to create temp data dir: %v", err)
		}
		tor.Debugf("Created temp data directory at: %v", tor.DataDir)
		tor.DeleteDataDirOnClose = !conf.RetainTempDataDir
	} else if err := os.MkdirAll(tor.DataDir, 0600); err != nil {
		return nil, fmt.Errorf("Unable to create data dir: %v", err)
	}
	// From this point on, we must close tor if we error
	// Start tor
	err := tor.startProcess(ctx, conf)
	// Connect the controller
	if err == nil {
		err = tor.connectController(ctx, conf)
	}
	// Attempt eager auth w/ no password
	if err == nil && !conf.DisableEagerAuth {
		err = tor.Control.Authenticate("")
	}
	// If there was an error, we have to try to close here but it may leave the process open
	if err != nil {
		if closeErr := tor.Close(); closeErr != nil {
			err = fmt.Errorf("Error on start: %v (also got error trying to close: %v)", err, closeErr)
		}
	}
	return tor, err
}

func (t *Tor) startProcess(ctx context.Context, conf *StartConf) error {
	// Get the creator
	var creator process.Creator
	if conf.Embedded {
		return fmt.Errorf("Embedded Tor not yet supported")
	} else {
		torPath := conf.ExePath
		if torPath == "" {
			torPath = "tor"
		}
		creator = process.NewCreator(torPath)
	}
	// Build the args
	args := []string{"--DataDirectory", t.DataDir}
	if !conf.DisableCookieAuth {
		args = append(args, "--CookieAuthentication", "1")
	}
	if !conf.EnableNetwork {
		args = append(args, "--DisableNetwork", "1")
	}
	// If there is no Torrc file, create a blank temp one
	torrcFileName := conf.TorrcFile
	if torrcFileName == "" {
		torrcFile, err := ioutil.TempFile(t.DataDir, "torrc-")
		if err != nil {
			return err
		}
		torrcFileName = torrcFile.Name()
		if err = torrcFile.Close(); err != nil {
			return err
		}
	}
	args = append(args, "-f", torrcFileName)
	// Create file for Tor to write the control port to if it's not told to us
	var controlPortFileName string
	var err error
	if conf.ControlPort == 0 {
		controlPortFile, err := ioutil.TempFile(t.DataDir, "control-port-")
		if err != nil {
			return err
		}
		controlPortFileName = controlPortFile.Name()
		if err = controlPortFile.Close(); err != nil {
			return err
		}
		args = append(args, "--ControlPort", "auto", "--ControlPortWriteToFile", controlPortFile.Name())
	}
	// Start process with the args
	var processCtx context.Context
	processCtx, t.ProcessCancelFunc = context.WithCancel(ctx)
	args = append(args, conf.ExtraArgs...)
	p, err := creator.New(processCtx, args...)
	if err != nil {
		return err
	}
	t.Debugf("Starting tor with args %v", args)
	if err = p.Start(); err != nil {
		return err
	}
	t.Process = p
	// Try a few times to read the control port file if we need to
	t.ControlPort = conf.ControlPort
	if t.ControlPort == 0 {
	ControlPortCheck:
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				break ControlPortCheck
			default:
				// Try to read the controlport file, or wait a bit
				var byts []byte
				if byts, err = ioutil.ReadFile(controlPortFileName); err != nil {
					break ControlPortCheck
				} else if t.ControlPort, err = process.ControlPortFromFileContents(string(byts)); err == nil {
					break ControlPortCheck
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
		if err != nil {
			return fmt.Errorf("Unable to read control port file: %v", err)
		}
	}
	return nil
}

func (t *Tor) connectController(ctx context.Context, conf *StartConf) error {
	t.Debugf("Connecting to control port %v", t.ControlPort)
	textConn, err := textproto.Dial("tcp", "127.0.0.1:"+strconv.Itoa(t.ControlPort))
	if err != nil {
		return err
	}
	t.Control = control.NewConn(textConn)
	t.Control.DebugWriter = t.DebugWriter
	return nil
}

func (t *Tor) Close() error {
	errs := []error{}
	// If controller is authenticated, send the quit signal to the process. Otherwise, just close the controller.
	sentHalt := false
	if t.Control != nil {
		if t.Control.Authenticated {
			if err := t.Control.Signal("HALT"); err != nil {
				errs = append(errs, fmt.Errorf("Unable to signal halt: %v", err))
			} else {
				sentHalt = true
			}
		}
		// Now close the controller
		if err := t.Control.Close(); err != nil {
			errs = append(errs, fmt.Errorf("Unable to close contrlller: %v", err))
		} else {
			t.Control = nil
		}
	}
	if t.Process != nil {
		// If we didn't halt, we have to force kill w/ the cancel func
		if !sentHalt {
			t.ProcessCancelFunc()
		}
		// Wait for a bit to make sure it stopped
		errCh := make(chan error, 1)
		var waitErr error
		go func() { errCh <- t.Process.Wait() }()
		select {
		case waitErr = <-errCh:
			if waitErr != nil {
				errs = append(errs, fmt.Errorf("Process wait failed: %v", waitErr))
			}
		case <-time.After(300 * time.Millisecond):
			errs = append(errs, fmt.Errorf("Process did not exit after 300 ms"))
		}
		if waitErr == nil {
			t.Process = nil
		}
	}
	// Get rid of the entire data dir
	if t.DeleteDataDirOnClose {
		if err := os.RemoveAll(t.DataDir); err != nil {
			errs = append(errs, fmt.Errorf("Failed to remove data dir %v: %v", t.DataDir, err))
		}
	}
	// Combine the errors if present
	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("Got %v errors while closing - %v", len(errs), errs)
}
