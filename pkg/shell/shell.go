// Package shell simplifies executing a command line utility, like kubectl, or a shell command.
package shell

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/giantswarm/microerror"
)

// RunCommand executes a command with the given arguments and environment variables.
// It returns the output, the exit code and and error.
func RunCommand(ctx context.Context, name string, envVars []string, arg ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, name, arg...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	cmd.Env = append(os.Environ(), envVars...)

	if err := cmd.Start(); err != nil {
		return out.String(), -1, microerror.Maskf(couldNotStartError, err.Error())
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return stderr.String(), status.ExitStatus(), microerror.Maskf(problemInExecutionError, exiterr.Error())
			}
		} else {
			return stderr.String(), -1, microerror.Maskf(problemInExecutionError, err.Error())
		}
	}

	return out.String(), 0, nil
}
