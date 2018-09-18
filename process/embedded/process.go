// Package embedded implements process interfaces for statically linked,
// embedded Tor. Note, processes created here are not killed when a context is
// done like w/ os.Exec.
//
// Usage
//
// This package can be used with CGO to statically compile Tor. This package
// expects https://github.com/cretz/tor-static to be cloned at
// $GOPATH/src/github.com/cretz/tor-static as if it was fetched with go get. To
// build the needed static libs, follow the README in that project. Once the
// static libs are built, this uses CGO to statically link them here. For
// Windows this means something like http://www.msys2.org/ needs to be
// installed with gcc.exe on the PATH (i.e. the same gcc that was used to build
// the static Tor lib).
package embedded

import (
	"context"
	"fmt"

	"github.com/cretz/bine/process"
)

/*
#cgo CFLAGS: -I${SRCDIR}/../../../tor-static/tor/src/or
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/tor/src/or -ltor
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/tor/src/common -lor -lor-crypto -lcurve25519_donna -lor-ctime -lor-event
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/tor/src/trunnel -lor-trunnel
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/tor/src/ext/keccak-tiny -lkeccak-tiny
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/tor/src/ext/ed25519/ref10 -led25519_ref10
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/tor/src/ext/ed25519/donna -led25519_donna
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/libevent/dist/lib -levent
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/xz/dist/lib -llzma
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/zlib/dist/lib -lz
#cgo LDFLAGS: -L${SRCDIR}/../../../tor-static/openssl/dist/lib -lssl -lcrypto
#cgo windows LDFLAGS: -lws2_32 -lcrypt32 -lgdi32
#cgo !windows LDFLAGS: -lm

#include <stdlib.h>
#include <tor_api.h>

// Ref: https://stackoverflow.com/questions/45997786/passing-array-of-string-as-parameter-from-go-to-c-function

static char** makeCharArray(int size) {
	return calloc(sizeof(char*), size);
}

static void setArrayString(char **a, char *s, int n) {
	a[n] = s;
}

static void freeCharArray(char **a, int size) {
	int i;
	for (i = 0; i < size; i++)
		free(a[i]);
	free(a);
}
*/
import "C"

type embeddedCreator struct{}

// NewCreator creates a process.Creator for statically-linked Tor embedded in
// the binary.
func NewCreator() process.Creator {
	return embeddedCreator{}
}

type embeddedProcess struct {
	ctx    context.Context
	args   []string
	doneCh chan int
}

func (embeddedCreator) New(ctx context.Context, args ...string) (process.Process, error) {
	return &embeddedProcess{ctx: ctx, args: args}, nil
}

func (e *embeddedProcess) Start() error {
	if e.doneCh != nil {
		return fmt.Errorf("Already started")
	}
	// Create the char array for the args
	args := append([]string{"tor"}, e.args...)
	charArray := C.makeCharArray(C.int(len(args)))
	for i, a := range args {
		C.setArrayString(charArray, C.CString(a), C.int(i))
	}
	// Build the conf
	conf := C.tor_main_configuration_new()
	if code := C.tor_main_configuration_set_command_line(conf, C.int(len(args)), charArray); code != 0 {
		C.tor_main_configuration_free(conf)
		C.freeCharArray(charArray, C.int(len(args)))
		return fmt.Errorf("Failed to set command line args, code: %v", int(code))
	}
	// Run it async
	e.doneCh = make(chan int, 1)
	go func() {
		defer C.freeCharArray(charArray, C.int(len(args)))
		defer C.tor_main_configuration_free(conf)
		e.doneCh <- int(C.tor_run_main(conf))
	}()
	return nil
}

func (e *embeddedProcess) Wait() error {
	if e.doneCh == nil {
		return fmt.Errorf("Not started")
	}
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case code := <-e.doneCh:
		if code == 0 {
			return nil
		}
		return fmt.Errorf("Command completed with error exit code: %v", code)
	}
}
