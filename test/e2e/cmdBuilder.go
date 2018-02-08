package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type cmdBuilder struct {
	cmd     *exec.Cmd
	timeout <-chan time.Time
}

func (b cmdBuilder) Exec() (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := b.cmd
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	log.Printf("Running '%s %s'", cmd.Path, strings.Join(cmd.Args[1:], " ")) // skip arg[0] as it is printed separately
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("Error starting %v: \n Command stdout: \n %v \n stderr: \n %v \n error: \n %v \n", cmd, cmd.Stdout, cmd.Stderr, err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("Error running %v:\nCommand stdout: \n %v \n stderr: \n %v \n error: \n %v \n", cmd, cmd.Stdout, cmd.Stderr, err)
		}
	case <-b.timeout:
		b.cmd.Process.Kill()
		return "", fmt.Errorf("Timed out waiting for command %v: \n Command stdout: \n %v \n stderr: \n %v \n", cmd, cmd.Stdout, cmd.Stderr)
	}
	if stderr.String() != "" {
		log.Printf("stderr: %q", stderr.String())
	}
	return stdout.String(), nil
}
