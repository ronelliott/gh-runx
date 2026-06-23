//go:build !windows

// Package runner replaces the current process with a downloaded program.
package runner

import (
	"fmt"
	"os"
	"syscall"
)

// Exec replaces the current process with the program at path, forwarding args
// and the current environment. On success it does not return.
func Exec(path string, args []string) error {
	argv := append([]string{path}, args...)
	if err := syscall.Exec(path, argv, os.Environ()); err != nil {
		return fmt.Errorf("runner: executing %s: %w", path, err)
	}
	return nil
}
