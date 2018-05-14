package process

import (
	"context"
	"os"
	"os/exec"
)

// Creator is the interface for process creation.
type Creator interface {
	New(ctx context.Context, args ...string) (Process, error)
}

type exeProcessCreator struct {
	exePath string
}

// NewCreator creates a Creator for external Tor process execution based on the given exe path.
func NewCreator(exePath string) Creator {
	return &exeProcessCreator{exePath}
}

func (e *exeProcessCreator) New(ctx context.Context, args ...string) (Process, error) {
	cmd := exec.CommandContext(ctx, e.exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}
