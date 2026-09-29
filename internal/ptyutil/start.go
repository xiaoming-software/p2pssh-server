//go:build (darwin || linux) && cgo

package ptyutil

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func startWithFds(cmd *exec.Cmd, masterFd, slaveFd int) (*os.File, error) {
	master := os.NewFile(uintptr(masterFd), "/dev/ptmx")
	slave := os.NewFile(uintptr(slaveFd), "pts")
	if master == nil || slave == nil {
		_ = syscall.Close(masterFd)
		_ = syscall.Close(slaveFd)
		return nil, fmt.Errorf("pty fd wrap failed")
	}

	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	// Ctty must be the child's fd index (stdin=0), not the parent's slaveFd.
	// Otherwise Go reports: "Setctty set but Ctty not valid in child".
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}
	if err := cmd.Start(); err != nil {
		_ = master.Close()
		_ = slave.Close()
		return nil, err
	}
	_ = slave.Close()
	return master, nil
}
