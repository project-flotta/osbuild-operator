package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const (
	DefaultTimeout = 10 * time.Second
)

type commandResult struct {
	cmd      string
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
	exitcode int
}

func (cmd commandResult) Failed() bool {
	return cmd.exitcode != 0
}

func (cmd commandResult) GetErrorMessage() string {
	return fmt.Sprintf(
		"failed to execute '%s', with exitcode: '%d' and stdout output %s stderr: %s",
		cmd.cmd, cmd.exitcode, cmd.GetStdout(), cmd.GetStderr())
}

func (cmd commandResult) GetStdout() string {
	return cmd.stdout.String()
}

func (cmd commandResult) GetStderr() string {
	return cmd.stderr.String()
}

func execCommandWithTimeout(cmd string, timeout time.Duration) *commandResult {
	res := &commandResult{
		cmd:    cmd,
		stdout: new(bytes.Buffer),
		stderr: new(bytes.Buffer),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	execCmd := exec.CommandContext(ctx, "bash", "-c", res.cmd) //nolint:gosec
	execCmd.Stderr = res.stderr
	execCmd.Stdout = res.stdout

	err := execCmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			res.exitcode = exitError.ExitCode()
		}
	}
	return res
}

func execCommand(cmd string, args ...interface{}) *commandResult {
	command := cmd
	if len(args) > 0 {
		command = fmt.Sprintf(cmd, args...)
	}

	return execCommandWithTimeout(command, DefaultTimeout)
}
