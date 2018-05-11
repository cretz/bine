package process

import (
	"context"
	"os"
	"os/exec"
)

type ProcessCreator interface {
	New(ctx context.Context, args ...string) (Process, error)
}

type exeProcessCreator struct {
	exePath string
}

func NewProcessCreator(exePath string) ProcessCreator {
	return &exeProcessCreator{exePath}
}
func (e *exeProcessCreator) New(ctx context.Context, args ...string) (Process, error) {
	proc := &exeProcess{Cmd: exec.CommandContext(ctx, e.exePath, args...)}
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc, nil
}
