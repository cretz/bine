// Package embedded implements process interfaces for statically linked,
// embedded Tor.
//
// TODO: not finished yet
package embedded

import "github.com/cretz/bine/process"

// NewCreator creates a process.Creator for statically-linked Tor embedded in
// the binary.
func NewCreator() process.Creator {
	panic("TODO: embedding not implemented yet")
}
