package process

import (
	"context"
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
	return &exeProcess{Cmd: exec.CommandContext(ctx, e.exePath, args...)}, nil
}
