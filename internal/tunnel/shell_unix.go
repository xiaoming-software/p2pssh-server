//go:build unix

package tunnel

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/xiaoming-software/p2pssh-server/internal/ptyutil"
)

func startShell(cols, rows uint16) (io.ReadWriteCloser, func(cols, rows uint16) error, func(), error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "darwin" {
			shell = "/bin/zsh"
		} else {
			shell = "/bin/bash"
		}
	}
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	cmd := exec.Command(shell, "-l")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		fmt.Sprintf("COLUMNS=%d", cols),
		fmt.Sprintf("LINES=%d", rows),
	)
	f, err := ptyutil.StartWithSize(cmd, rows, cols)
	if err != nil {
		return nil, nil, nil, err
	}
	resize := func(c, r uint16) error {
		if c == 0 {
			c = 80
		}
		if r == 0 {
			r = 24
		}
		return ptyutil.Setsize(f, r, c)
	}
	cleanup := func() {
		_ = f.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}
	return f, resize, cleanup, nil
}
