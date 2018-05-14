package process

// Process is the interface implemented by Tor processes.
type Process interface {
	// Start starts the Tor process in the background and returns. It is analagous to os/exec.Cmd.Start.
	Start() error
	// Wait waits for the Tor process to exit and returns error if it was not a successful exit.  It is analagous to
	// os/exec.Cmd.Wait.
	Wait() error
}
