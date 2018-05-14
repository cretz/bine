package tor

import "fmt"

func (t *Tor) DebugEnabled() bool {
	return t.DebugWriter != nil
}

func (t *Tor) Debugf(format string, args ...interface{}) {
	if w := t.DebugWriter; w != nil {
		fmt.Fprintf(w, format+"\n", args...)
	}
}
