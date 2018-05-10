package process

import (
	"context"
	"os/exec"
)

type Tor interface {
	Start(ctx context.Context, args []string) error
}

type exeTor struct {
	exePath string
}

func FromExePath(exePath string) Tor {
	return &exeTor{exePath}
}

func (e *exeTor) Start(ctx context.Context, args []string) error {
	return exec.CommandContext(ctx, e.exePath, args...).Start()
}
