package main

import (
	"os/exec"
)

type kubernikusctlBuilder struct {
	cmdBuilder
}

func NewKubernikusctlCommand(args ...string) *kubernikusctlBuilder {
	b := new(kubernikusctlBuilder)
	b.cmd = KubernikusctlCmd(args...)
	return b
}

func RunKubernikusctlHostCmd(args ...string) (string, error) {
	return NewKubernikusctlCommand(args...).Exec()
}

func KubernikusctlCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(KubernikusctlBinaryName, args...)
	return cmd
}
