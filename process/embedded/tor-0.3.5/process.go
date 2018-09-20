// Package tor035 implements process interfaces for statically linked
// Tor 0.3.5.x versions. See the process/embedded package for the generic
// abstraction
package tor035

import (
	"context"
	"fmt"

	"github.com/cretz/bine/process"
)

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../tor-static/tor/src/feature/api
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor/src/core -ltor-app
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor/src/lib -ltor-compress -ltor-evloop -ltor-tls -ltor-crypt-ops -lcurve25519_donna -ltor-process -ltor-time -ltor-fs -ltor-encoding -ltor-sandbox -ltor-container -ltor-net -ltor-thread -ltor-memarea -ltor-math -ltor-meminfo -ltor-osinfo -ltor-log -ltor-lock -ltor-fdio -ltor-string -ltor-term -ltor-smartlist-core -ltor-malloc -ltor-wallclock -ltor-err -ltor-intmath -ltor-ctime -ltor-trace
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor/src/ext/keccak-tiny -lkeccak-tiny
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor/src/ext/ed25519/ref10 -led25519_ref10
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor/src/ext/ed25519/donna -led25519_donna
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor/src/trunnel -lor-trunnel
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/libevent/dist/lib -levent
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/xz/dist/lib -llzma
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/zlib/dist/lib -lz
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/openssl/dist/lib -lssl -lcrypto
#cgo windows LDFLAGS: -lws2_32 -lcrypt32 -lgdi32 -liphlpapi
#cgo !windows LDFLAGS: -lm

#include <stdlib.h>
#ifdef _WIN32
	#include <winsock2.h>
#endif
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

// ProcessCreator implements process.Creator
type ProcessCreator struct {
	// If set to true, ProcessControlSocket will have a raw socket to
	// communicate with Tor on.
	SetupControlSocket bool
}

// ProviderVersion returns the Tor provider name and version exposed from the
// Tor embedded API.
func ProviderVersion() string {
	return C.GoString(C.tor_api_get_provider_version())
}

// NewProcessCreator creates a process.Creator for statically-linked Tor
// embedded in the binary.
func NewProcessCreator() *ProcessCreator {
	return &ProcessCreator{}
}

type embeddedProcess struct {
	ctx           context.Context
	mainConf      *C.struct_tor_main_configuration_t
	controlSocket uintptr
	args          []string
	doneCh        chan int
}

// New implements process.Creator.New
func (p *ProcessCreator) New(ctx context.Context, args ...string) (process.Process, error) {
	ret := &embeddedProcess{
		ctx:      ctx,
		mainConf: C.tor_main_configuration_new(),
		args:     args,
	}
	// If they want a control socket, this is where we add it
	if p.SetupControlSocket {
		ret.controlSocket = uintptr(C.tor_main_configuration_setup_control_socket(ret.mainConf))
	}
	return ret, nil
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
	if code := C.tor_main_configuration_set_command_line(e.mainConf, C.int(len(args)), charArray); code != 0 {
		C.tor_main_configuration_free(e.mainConf)
		C.freeCharArray(charArray, C.int(len(args)))
		return fmt.Errorf("Failed to set command line args, code: %v", int(code))
	}
	// Run it async
	e.doneCh = make(chan int, 1)
	go func() {
		defer C.freeCharArray(charArray, C.int(len(args)))
		defer C.tor_main_configuration_free(e.mainConf)
		e.doneCh <- int(C.tor_run_main(e.mainConf))
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

// ProcessControlSocket returns a non-zero value for a process created by a
// ProcessCreator with SetupControlSocket as true. Note, the value of this is
// invalid when Start returns.
func ProcessControlSocket(p process.Process) uintptr {
	return p.(*embeddedProcess).controlSocket
}
