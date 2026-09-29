//go:build !cgo || (!darwin && !linux)

package ptyutil

import (
	"fmt"
	"os"
	"os/exec"
)

// StartWithSize is unavailable without cgo on unix targets.
func StartWithSize(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	return nil, fmt.Errorf("pty requires cgo on darwin/linux")
}

// Setsize is unavailable without cgo on unix targets.
func Setsize(f *os.File, rows, cols uint16) error {
	return fmt.Errorf("pty requires cgo on darwin/linux")
}
