package process

import (
	"os/exec"
)

type Process interface {
	Run() error
}

type exeProcess struct {
	*exec.Cmd
}
