package controltest

import (
	"flag"
	"testing"
)

func SkipIfNotRunningSpecifically(t *testing.T) {
	if f := flag.Lookup("test.run"); f == nil || f.Value == nil || f.Value.String() != t.Name() {
		t.Skip("Only runs if -run specifies this test exactly")
	}
}
