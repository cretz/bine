package tor

import (
	"context"
	"io"
	"net"
)

type Tor struct {
}

type StartConf struct {
	// TODO: docs...Nil means contet.Background
	Context context.Context
	// TODO: docs...Empty string means just "tor" either locally or on PATH
	ExePath string
	// TODO: docs...If true, doesn't use exe path, uses statically compiled Tor
	Embedded bool
	// TODO: docs...If 0, Tor is asked to store the control port in a temporary file in the data directory that is
	// deleted after read
	ControlPort int
	//  TODO: docs...If not empty, this is the data directory used and *TempDataDir* fields are unused
	DataDir string
	//  TODO: docs...If not empty, this is the parent directory that a child dir is created for data. If empty, the
	// current dir is assumed. This has no effect if DataDir is set.
	TempDataDirBase string
	//  TODO: docs...If true the temporary data dir is not deleted on close. This has no effect if DataDir is set.
	RetainTempDataDir bool
	//  TODO: docs...Any extra CLI arguments to pass to Tor. This are applied after other CLI args.
	ExtraArgs []string
	// TODO: docs...
	DebugWriter io.Writer
}

func (t *Tor) Start(conf *StartConf) error {
	// actualConf := *conf
	panic("TODO")
}

type OnionConf struct {
	Port       int
	TargetPort int
}

func (t *Tor) Listen(conf *OnionConf) (net.Listener, error) {
	panic("TODO")
}
