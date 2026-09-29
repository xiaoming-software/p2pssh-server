//go:build darwin && cgo

package ptyutil

/*
#include <util.h>
#include <sys/ioctl.h>
#include <unistd.h>
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// StartWithSize starts cmd on a new PTY.
func StartWithSize(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	var master, slave C.int
	var win C.struct_winsize
	win.ws_row = C.ushort(rows)
	win.ws_col = C.ushort(cols)
	if C.openpty(&master, &slave, nil, nil, &win) != 0 {
		return nil, fmt.Errorf("openpty failed")
	}
	return startWithFds(cmd, int(master), int(slave))
}

// Setsize updates PTY window size.
func Setsize(f *os.File, rows, cols uint16) error {
	var win C.struct_winsize
	win.ws_row = C.ushort(rows)
	win.ws_col = C.ushort(cols)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(C.TIOCSWINSZ), uintptr(unsafe.Pointer(&win)))
	if errno != 0 {
		return errno
	}
	return nil
}
