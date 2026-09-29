//go:build windows

package tunnel

import (
	"fmt"
	"io"
)

func startShell(cols, rows uint16) (io.ReadWriteCloser, func(cols, rows uint16) error, func(), error) {
	return nil, nil, nil, fmt.Errorf("shell PTY not supported on windows server; install OpenSSH and use TCP forward, or run server on Linux/macOS")
}
