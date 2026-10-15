//go:build darwin || linux

package ptyutil

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// StartWithSize starts cmd on a new PTY with the given window size.
// Pure Go (no cgo), so darwin/linux binaries can be cross-built from macOS.
func StartWithSize(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	return pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
}

// Setsize updates PTY window size.
func Setsize(f *os.File, rows, cols uint16) error {
	return pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: cols})
}
