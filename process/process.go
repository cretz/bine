package process

import (
	"os/exec"
)

type Process interface {
	Start() error
	Wait() error
}

type exeProcess struct {
	*exec.Cmd
}
