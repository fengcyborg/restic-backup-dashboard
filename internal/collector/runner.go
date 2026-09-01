package collector

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

const maxCommandOutput = 4 << 20
const commandTimeout = 15 * time.Second

type Runner interface {
	Run(context.Context, []string) (string, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("empty command")
	}
	commandContext, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	var stdout bytes.Buffer
	command := exec.CommandContext(commandContext, argv[0], argv[1:]...)
	command.Stdout = &stdout
	command.Stderr = io.Discard
	err := command.Run()
	if stdout.Len() > maxCommandOutput {
		return "", errors.New("command output exceeds safety limit")
	}
	return stdout.String(), err
}
