package main

import (
	"fmt"
	"os/exec"
)

type kubectlBuilder struct {
	cmdBuilder
}

func NewKubectlCommand(args ...string) *kubectlBuilder {
	b := new(kubectlBuilder)
	b.cmd = KubectlCmd(args...)
	return b
}

func RunKubectlHostCmd(namespace, name, cmd string) (string, error) {
	return RunKubectl("exec", fmt.Sprintf("--namespace=%v", namespace), name, "--", "/bin/sh", "-c", cmd)
}

func RunKubectl(args ...string) (string, error) {
	return NewKubectlCommand(args...).Exec()
}

func KubectlCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("kubectl", args...)
	return cmd
}
