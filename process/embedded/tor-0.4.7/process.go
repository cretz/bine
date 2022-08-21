// Package tor047 implements process interfaces for statically linked
// Tor 0.4.7.x versions. See the process/embedded package for the generic
// abstraction
package tor047

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/cretz/bine/process"
)

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../tor-static/tor/src/feature/api
// The libs below are generated via tor-static's show-libs
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/tor -ltor
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/libevent/dist/lib -levent
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/xz/dist/lib -llzma
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/zlib/dist/lib -lz
#cgo LDFLAGS: -L${SRCDIR}/../../../../tor-static/openssl/dist/lib -lssl -lcrypto
#cgo windows LDFLAGS: -lws2_32 -lcrypt32 -lgdi32 -liphlpapi -lshlwapi -Wl,-Bstatic -lpthread
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

type embeddedCreator struct{}

// ProviderVersion returns the Tor provider name and version exposed from the
// Tor embedded API.
func ProviderVersion() string {
	return C.GoString(C.tor_api_get_provider_version())
}

// NewCreator creates a process.Creator for statically-linked Tor embedded in
// the binary.
func NewCreator() process.Creator {
	return embeddedCreator{}
}

type embeddedProcess struct {
	ctx      context.Context
	mainConf *C.struct_tor_main_configuration_t
	args     []string
	doneCh   chan int
}

// New implements process.Creator.New
func (embeddedCreator) New(ctx context.Context, args ...string) (process.Process, error) {
	return &embeddedProcess{
		ctx: ctx,
		// TODO: mem leak if they never call Start; consider adding a Close()
		mainConf: C.tor_main_configuration_new(),
		args:     args,
	}, nil
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

func (e *embeddedProcess) EmbeddedControlConn() (net.Conn, error) {
	file := os.NewFile(uintptr(C.tor_main_configuration_setup_control_socket(e.mainConf)), "")
	conn, err := net.FileConn(file)
	if err != nil {
		err = fmt.Errorf("Unable to create conn from control socket: %v", err)
	}
	return conn, err
}
