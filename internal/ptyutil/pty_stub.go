//go:build !darwin && !linux

package ptyutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// StartWithSize is unavailable on this platform (Windows: use TCP forward to OpenSSH).
func StartWithSize(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	return nil, fmt.Errorf("shell PTY not supported on %s", runtime.GOOS)
}

// Setsize is unavailable on this platform.
func Setsize(f *os.File, rows, cols uint16) error {
	return fmt.Errorf("shell PTY not supported on %s", runtime.GOOS)
}
