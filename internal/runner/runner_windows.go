//go:build windows

// Package runner runs a downloaded program as a child process on Windows, where
// process replacement via syscall.Exec is unavailable.
package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Exec runs the program at path as a child process, forwarding args, the current
// environment, and standard streams, then exits with the child's exit code. On
// success it does not return.
func Exec(path string, args []string) error {
	cmd := exec.Command(path, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("runner: executing %s: %w", path, err)
	}
	os.Exit(0)
	return nil
}
